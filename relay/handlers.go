package relay

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/nbd-wtf/go-nostr"

	"github.com/saveblush/reraw-relay/core/cctx"
	"github.com/saveblush/reraw-relay/core/config"
	"github.com/saveblush/reraw-relay/core/generic"
	"github.com/saveblush/reraw-relay/core/utils"
	"github.com/saveblush/reraw-relay/core/utils/logger"
	"github.com/saveblush/reraw-relay/models"
	"github.com/saveblush/reraw-relay/pgk/eventstore"
	"github.com/saveblush/reraw-relay/pgk/nips/nip09"
	"github.com/saveblush/reraw-relay/pgk/nips/nip13"
	"github.com/saveblush/reraw-relay/pgk/nips/nip40"
	"github.com/saveblush/reraw-relay/pgk/nips/nip45"
)

type service struct {
	config *config.Configs
	cctx   *cctx.Context
	ctx    context.Context
	muRes  sync.Mutex

	eventstore eventstore.Service
	nip09      nip09.Service
	nip13      nip13.Service
	nip40      nip40.Service
	nip45      nip45.Service

	Conn         *Conn
	StoreEvent   StoreEvent
	RejectFilter RejectFilter
	RejectEvent  RejectEvent
}

// newHandleEvent new handle event
func newHandleEvent() *service {
	return &service{
		config:     config.CF,
		cctx:       &cctx.Context{},
		ctx:        context.TODO(),
		eventstore: eventstore.NewService(),
		nip09:      nip09.NewService(),
		nip13:      nip13.NewService(),
		nip40:      nip40.NewService(),
		nip45:      nip45.NewService(),
	}
}

// handleEvent handle event
func (s *service) handleEvent(msg []byte) error {
	start := utils.Now()
	envelope := nostr.ParseMessage(msg)
	if envelope == nil {
		_ = s.responseError(errInvalidMessage.Error())
		return errInvalidMessage
	}
	//logger.Log.Info("parse msg: ", envelope)

	switch env := envelope.(type) {
	case *nostr.EventEnvelope:
		err := s.onEvent(&env.Event)
		if err != nil {
			logger.Log.Errorf("[event] error: %s", err)
			return err
		}
		logger.Log.Info("[event] processed in ", time.Since(start))

	case *nostr.ReqEnvelope:
		err := s.onReq(env.SubscriptionID, &env.Filters)
		if err != nil {
			logger.Log.Errorf("[req] error: %s", err)
			return err
		}
		logger.Log.Info("[req] processed in ", time.Since(start))

	case *nostr.CloseEnvelope:
		err := s.onClose(env)
		if err != nil {
			logger.Log.Errorf("[close] error: %s", err)
			return err
		}
		logger.Log.Info("[close] processed in ", time.Since(start))

	case *nostr.CountEnvelope:
		err := s.onCount(env.SubscriptionID, &env.Filters)
		if err != nil {
			logger.Log.Errorf("[count] error: %s", err)
			return err
		}
		logger.Log.Info("[count] processed in ", time.Since(start))

	default:
		_ = s.responseError(errUnknownCommand.Error())
		return errUnknownCommand
	}

	return nil
}

func (s *service) onEvent(evt *nostr.Event) error {
	// check reject
	for _, rejectFunc := range s.RejectEvent {
		if reject, msg := rejectFunc(s.cctx, evt); reject {
			_ = s.responseOK(evt.ID, false, msg)
			return errors.New(msg)
		}
	}

	// clear older
	err := s.clearEventOlder(evt)
	if err != nil {
		logger.Log.Errorf("clear older error: %s", err)
		_ = s.responseOK(evt.ID, false, err.Error())
		return err
	}

	// ckeck duplicate
	fetch, err := s.eventstore.FindByID(s.cctx, evt.ID)
	if err != nil {
		logger.Log.Errorf("find duplicate error: %s", err)
		_ = s.responseOK(evt.ID, false, errConnectDatabase.Error())
		return errConnectDatabase
	}
	if !generic.IsEmpty(fetch) {
		_ = s.responseOK(evt.ID, true, errDuplicate.Error())
		return errDuplicate
	}

	// store event
	for _, storeFunc := range s.StoreEvent {
		err := storeFunc(s.cctx, evt)
		if err != nil {
			logger.Log.Errorf("func store event error: %s", err)
			_ = s.responseOK(evt.ID, false, nostr.NormalizeOKMessage(err.Error(), "error"))
			return err
		}
	}

	err = s.storeEvent(evt)
	if err != nil {
		logger.Log.Errorf("store event error: %s", err)
		_ = s.responseOK(evt.ID, false, errConnectDatabase.Error())
		return errConnectDatabase
	}

	// handlers kind
	switch evt.Kind {
	case 5:
		// soft delete
		err = s.nip09.CancelEvent(s.cctx, evt)
		if err != nil {
			logger.Log.Errorf("soft delete error: %s", err)
			_ = s.responseOK(evt.ID, false, errConnectDatabase.Error())
			return errConnectDatabase
		}
	}

	_ = s.responseOK(evt.ID, true, "")

	return nil
}

func (s *service) onReq(subID string, filters *nostr.Filters) error {
	for idx, filter := range *filters {
		// check reject
		for _, rejectFunc := range s.RejectFilter {
			if reject, msg := rejectFunc(&filter); reject {
				_ = s.responseClosed(subID, msg)
				return errors.New(msg)
			}
		}

		fetch, err := s.eventstore.FindAll(s.cctx, &eventstore.Request{NostrFilter: &filter})
		if err != nil {
			logger.Log.Errorf("find filter [index: %d] error: %s", idx, err)
			_ = s.responseClosed(subID, errConnectDatabase.Error())
			return err
		}

		for _, v := range fetch {
			_ = s.responseEvent(subID, v)
		}

		_ = s.responseEose(subID)
	}

	return nil
}

func (s *service) onClose(env interface{}) error {
	subID, err := s.subID(env)
	if err != nil {
		_ = s.responseError(err.Error())
		return err
	}

	if generic.IsEmpty(subID) {
		_ = s.responseError(errSubIDNotFound.Error())
		return errSubIDNotFound
	}

	return nil
}

func (s *service) onCount(subID string, filters *nostr.Filters) error {
	for idx, filter := range *filters {
		count, err := s.nip45.CountEvent(s.cctx, &filter)
		if err != nil {
			logger.Log.Errorf("count filter [index: %d] error: %s", idx, err)
			_ = s.responseClosed(subID, errConnectDatabase.Error())
			return err
		}

		err = s.responseCount(subID, count)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *service) clearEventOlder(evt *nostr.Event) error {
	if generic.IsEmpty(evt) {
		return errors.New("invalid: event not found")
	}

	var isDeleteOlder bool
	filterEvent := &nostr.Filter{}
	if s.isEphemeralKind(evt.Kind) {
		// เหตุการณ์เกิดขึ้นชั่วคราว จะไม่จัดเก็บโดยรีเลย์
		return nil
	} else if s.isReplaceableKind(evt.Kind) {
		// event ที่แก้ไขข้อมูลได้
		filterEvent = &nostr.Filter{Authors: []string{evt.PubKey}, Kinds: []int{evt.Kind}}
		isDeleteOlder = true
	} else if s.isParamReplaceableKind(evt.Kind) {
		// NIP-33
		// เหตุการณ์ที่สามารถแทนที่ด้วยพารามิเตอร์ได้
		d := evt.Tags.GetFirst([]string{"d", ""})
		if d == nil {
			return errors.New("invalid: missing 'd' tag on parameterized replaceable event")
		}
		filterEvent = &nostr.Filter{Authors: []string{evt.PubKey}, Kinds: []int{evt.Kind}, Tags: nostr.TagMap{"d": []string{d.Value()}}}
		isDeleteOlder = true
	}

	// ลบข้อมูลเดิมก่อนยิงใหม่
	if isDeleteOlder && generic.IsEmpty(filterEvent) {
		return errors.New("invalid: missing 'filter' tag on parameterized replaceable event")
	}

	if isDeleteOlder {
		fetch, err := s.eventstore.FindAll(s.cctx, &eventstore.Request{NostrFilter: filterEvent})
		if err != nil {
			logger.Log.Errorf("find error: %s", err)
			return err
		}

		// ลบ event ที่เก่ากว่า
		for _, previous := range fetch {
			if s.isOlder(previous, evt) {
				err := s.eventstore.Delete(s.cctx, &models.RelayEvent{ID: previous.ID})
				if err != nil {
					logger.Log.Errorf("delete older error: %s", err)
					return err
				}
			}
		}
	}

	return nil
}

func (s *service) storeEvent(evt *nostr.Event) error {
	evtAddon := &models.EventAddon{}
	evtAddon.UpdatedIP = s.Conn.IP()

	// get expiration
	expiration, err := s.nip40.Expiration(s.cctx, evt)
	if err != nil {
		_ = s.responseOK(evt.ID, false, err.Error())
		return err
	}
	if !generic.IsEmpty(expiration) {
		evtAddon.Expiration = nostr.Timestamp(expiration)
	}

	v := &models.RelayEvent{
		ID:         evt.ID,
		CreatedAt:  evt.CreatedAt,
		Pubkey:     evt.PubKey,
		Kind:       evt.Kind,
		Content:    evt.Content,
		Tags:       evt.Tags,
		Sig:        evt.Sig,
		Expiration: evtAddon.Expiration,
		UpdatedIP:  evtAddon.UpdatedIP,
		UpdatedAt:  evtAddon.UpdatedAt,
		DeletedAt:  evtAddon.DeletedAt,
	}
	err = s.eventstore.Insert(s.cctx, v)
	if err != nil {
		logger.Log.Errorf("insert error: %s", err)
		return err
	}

	return nil
}

package relay

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/goccy/go-json"

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
	//respMutex sync.Mutex

	eventstore eventstore.Service
	nip09      nip09.Service
	nip13      nip13.Service
	nip40      nip40.Service
	nip45      nip45.Service

	client *Client

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
	var req []*json.RawMessage
	var cmd string

	start := utils.Now()
	defer func() { logger.Log.Infof("[%s] processed in %s", cmd, time.Since(start)) }()

	if err := json.Unmarshal(msg, &req); err != nil {
		_ = s.responseError(errInvalidMessage.Error())
		return errInvalidMessage
	}

	if len(req) < 2 {
		_ = s.responseError(errInvalidParamsMessage.Error())
		return errInvalidParamsMessage
	}

	json.Unmarshal(*req[0], &cmd)

	switch cmd {
	case "EVENT":
		err := s.onEvent(req)
		if err != nil {
			logger.Log.Errorf("[event] error: %s", err)
			return err
		}

	case "REQ":
		err := s.onReq(req)
		if err != nil {
			logger.Log.Errorf("[req] error: %s", err)
			return err
		}

	case "CLOSE":
		err := s.onClose(req)
		if err != nil {
			logger.Log.Errorf("[close] error: %s", err)
			return err
		}

	case "COUNT":
		err := s.onCount(req)
		if err != nil {
			logger.Log.Errorf("[count] error: %s", err)
			return err
		}

	default:
		_ = s.responseError(errUnknownCommand.Error())
		return errUnknownCommand
	}

	return nil
}

func (s *service) onEvent(req []*json.RawMessage) error {
	evt, err := s.parseEvent(req)
	if err != nil {
		return err
	}

	// check reject
	for _, rejectFunc := range s.RejectEvent {
		if reject, msg := rejectFunc(s.cctx, evt); reject {
			_ = s.responseOK(evt.ID, false, msg)
			return errors.New(msg)
		}
	}

	// clear older
	err = s.clearEventOlder(evt)
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
		_ = s.responseOK(evt.ID, true, errDuplicateEvent.Error())
		return errDuplicateEvent
	}

	// store event
	for _, storeFunc := range s.StoreEvent {
		err := storeFunc(s.cctx, evt)
		if err != nil {
			logger.Log.Errorf("func store event error: %s", err)
			_ = s.responseOK(evt.ID, false, fmt.Sprintf("error: %s", err))
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

func (s *service) onReq(req []*json.RawMessage) error {
	subID, err := s.subID(req)
	if err != nil {
		return err
	}

	filters, err := s.parseFilters(req)
	if err != nil {
		return err
	}

	for idx, filter := range *filters {
		// check reject
		for _, rejectFunc := range s.RejectFilter {
			if reject, msg := rejectFunc(&filter); reject {
				_ = s.responseClosed(subID, msg)
				return errors.New(msg)
			}
		}

		events, err := s.eventstore.FindAll(s.cctx, &eventstore.Request{NostrFilter: &filter})
		if err != nil {
			logger.Log.Errorf("find filter [index: %d] error: %s", idx, err)
			_ = s.responseClosed(subID, errConnectDatabase.Error())
			return err
		}

		for _, event := range events {
			_ = s.responseEvent(subID, event)
		}
	}

	_ = s.responseEose(subID)

	return nil
}

func (s *service) onClose(req []*json.RawMessage) error {
	subID, err := s.subID(req)
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

func (s *service) onCount(req []*json.RawMessage) error {
	subID, err := s.subID(req)
	if err != nil {
		return err
	}

	filters, err := s.parseFilters(req)
	if err != nil {
		return err
	}

	var total int64
	for idx, filter := range *filters {
		count, err := s.nip45.CountEvent(s.cctx, &filter)
		if err != nil {
			logger.Log.Errorf("count filter [index: %d] error: %s", idx, err)
			_ = s.responseClosed(subID, errConnectDatabase.Error())
			return err
		}
		total += *count
	}

	err = s.responseCount(subID, &total)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) clearEventOlder(evt *models.Event) error {
	if generic.IsEmpty(evt) {
		return errors.New("invalid: event not found")
	}

	filterEvent := &models.Filter{}
	var isDeleteOlder bool
	if s.isEphemeralKind(evt.Kind) {
		// เหตุการณ์เกิดขึ้นชั่วคราว จะไม่จัดเก็บโดยรีเลย์
		return nil
	} else if s.isReplaceableKind(evt.Kind) {
		// event ที่แก้ไขข้อมูลได้
		filterEvent = &models.Filter{Authors: []string{evt.Pubkey}, Kinds: []int{evt.Kind}}
		isDeleteOlder = true
	} else if s.isParamReplaceableKind(evt.Kind) {
		// NIP-33
		// เหตุการณ์ที่สามารถแทนที่ด้วยพารามิเตอร์ได้
		d := evt.Tags.FindKeyD()
		if d == "" {
			return errors.New("invalid: missing 'd' tag on parameterized replaceable event")
		}
		filterEvent = &models.Filter{Authors: []string{evt.Pubkey}, Kinds: []int{evt.Kind}, Tags: models.TagMap{"d": []string{d}}}
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
				err := s.eventstore.Delete(s.cctx, &models.Event{ID: previous.ID})
				if err != nil {
					logger.Log.Errorf("delete older error: %s", err)
					return err
				}
			}
		}
	}

	return nil
}

func (s *service) storeEvent(evt *models.Event) error {
	v := &models.Event{
		ID:        evt.ID,
		CreatedAt: models.Timestamp(evt.CreatedAt),
		Pubkey:    evt.Pubkey,
		Kind:      evt.Kind,
		Content:   evt.Content,
		Tags:      evt.Tags,
		Sig:       evt.Sig,
	}

	// get expiration
	expiration, err := s.nip40.Expiration(s.cctx, evt)
	if err != nil {
		_ = s.responseOK(evt.ID, false, err.Error())
		return err
	}
	if !generic.IsEmpty(expiration) {
		v.Expiration = expiration
	}

	err = s.eventstore.Insert(s.cctx, v)
	if err != nil {
		logger.Log.Errorf("insert error: %s", err)
		return err
	}

	return nil
}

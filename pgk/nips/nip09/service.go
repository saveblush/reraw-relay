package nip09

import (
	"errors"
	"strconv"

	"github.com/saveblush/reraw-relay/core/cctx"
	"github.com/saveblush/reraw-relay/core/config"
	"github.com/saveblush/reraw-relay/core/generic"
	"github.com/saveblush/reraw-relay/core/utils/logger"
	"github.com/saveblush/reraw-relay/models"
	"github.com/saveblush/reraw-relay/pgk/eventstore"
)

// Service service interface
type Service interface {
	CancelEvent(c *cctx.Context, evt *models.Event) error
}

type service struct {
	config     *config.Configs
	eventstore eventstore.Service
}

func NewService() Service {
	return &service{
		config:     config.CF,
		eventstore: eventstore.NewService(),
	}
}

// CancelEvent soft delete event
func (s *service) CancelEvent(c *cctx.Context, evt *models.Event) error {
	if generic.IsEmpty(evt) {
		return errors.New("invalid: event not found")
	}

	if evt.Pubkey == "" {
		return errors.New("invalid: missing 'pubkey' on parameterized deletion event")
	}

	// หา id event จาก tag "e"
	var ids []string
	tags := evt.Tags.FindAll("e")
	for _, v := range *tags {
		ids = append(ids, v.Value())
	}

	// หา kind จาก tag "k"
	var kinds []int
	tags = evt.Tags.FindAll("k")
	for _, v := range *tags {
		i, _ := strconv.Atoi(v.Value())
		kinds = append(kinds, i)
	}

	// find event
	filter := &models.Filter{IDs: ids, Kinds: kinds, Authors: []string{evt.Pubkey}}
	fetch, err := s.eventstore.FindAll(c, &eventstore.Request{NostrFilter: filter, NoLimit: true})
	if err != nil {
		logger.Log.Errorf("find event error: %s", err)
		return err
	}

	// cancel event
	for _, v := range fetch {
		err := s.eventstore.SoftDelete(c, &models.Event{ID: v.ID})
		if err != nil {
			logger.Log.Errorf("soft delete error: %s", err)
			return err
		}
	}

	return nil
}

package nip09

import (
	"errors"
	"strconv"
	"strings"

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

	filter := &models.Filter{}

	// หา event id จาก tag "e"
	var ids []string
	tags := evt.Tags.FindAll("e")
	for _, v := range *tags {
		ids = append(ids, v.Value())
	}

	if len(ids) > 0 {
		filter.IDs = ids
	} else {
		// หา filter จาก tag "a"
		tag := evt.Tags.FindFirst("a")
		if tag != nil && len(*tag) >= 2 {
			v := strings.Split(tag.Value(), ":")
			if len(v) == 3 {
				kind, _ := strconv.Atoi(v[0])
				author := v[1]
				identifier := v[2]
				filter.Kinds = []int{kind}
				filter.Authors = []string{author}
				filter.Tags = models.TagMap{"d": []string{identifier}}
				filter.Until = &evt.CreatedAt
			}
		}
	}

	if generic.IsEmpty(filter) {
		return errors.New("invalid: tags e or a not found")
	}

	if generic.IsEmpty(filter.Kinds) {
		// หา kind จาก tag "k"
		var kinds []int
		tags = evt.Tags.FindAll("k")
		for _, v := range *tags {
			i, _ := strconv.Atoi(v.Value())
			kinds = append(kinds, i)
		}
		if len(kinds) > 0 {
			filter.Kinds = kinds
		}
	}

	if generic.IsEmpty(filter) {
		return errors.New("invalid: filter not found")
	}

	// มองตามผู้สร้าง event
	filter.Authors = []string{evt.Pubkey}

	// find event
	fetch, err := s.eventstore.FindAll(c, &eventstore.Request{NostrFilter: filter, NoLimit: true})
	if err != nil {
		logger.Log.Errorf("find event error: %s", err)
		return errors.New("error: could not connect to the database")
	}

	// cancel event
	for _, v := range fetch {
		if v.Pubkey != evt.Pubkey {
			return errors.New("blocked: you are not the author of this event")
		}

		err := s.eventstore.SoftDelete(c, &models.Event{ID: v.ID})
		if err != nil {
			logger.Log.Errorf("soft delete error: %s", err)
			return errors.New("error: could not connect to the database")
		}
	}

	return nil
}

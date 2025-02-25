package nip45

import (
	"github.com/saveblush/reraw-relay/core/cctx"
	"github.com/saveblush/reraw-relay/core/config"
	"github.com/saveblush/reraw-relay/models"
	"github.com/saveblush/reraw-relay/pgk/eventstore"
)

// Service service interface
type Service interface {
	CountEvent(c *cctx.Context, req *models.Filter) (*int64, error)
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

func (s *service) CountEvent(c *cctx.Context, req *models.Filter) (*int64, error) {
	var noLimit bool
	if req.Limit == 0 {
		noLimit = true
	}

	res, err := s.eventstore.Count(c, &eventstore.Request{NostrFilter: req, DoCount: true, NoLimit: noLimit})
	if err != nil {
		return nil, err
	}

	return res, nil
}

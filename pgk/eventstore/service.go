package eventstore

import (
	"github.com/saveblush/reraw-relay/core/cctx"
	"github.com/saveblush/reraw-relay/core/config"
	"github.com/saveblush/reraw-relay/core/generic"
	"github.com/saveblush/reraw-relay/core/utils/logger"
	"github.com/saveblush/reraw-relay/models"
)

// Service service interface
type Service interface {
	FindAll(c *cctx.Context, req *Request) ([]*models.Event, error)
	FindByID(c *cctx.Context, ID string) (*models.Event, error)
	Count(c *cctx.Context, req *Request) (*int64, error)
	Insert(c *cctx.Context, req *models.Event) error
	SoftDelete(c *cctx.Context, req *models.Event) error
	Delete(c *cctx.Context, req *models.Event) error
	InsertBlacklist(c *cctx.Context, req *models.Blacklist) error
	FindBlacklists(c *cctx.Context, req *models.Blacklist) ([]*models.Blacklist, error)
	ClearEventsWithBlacklist(c *cctx.Context) error
}

type service struct {
	config     *config.Configs
	repository Repository
}

func NewService() Service {
	return &service{
		config:     config.CF,
		repository: NewRepository(),
	}
}

func (s *service) FindAll(c *cctx.Context, req *Request) ([]*models.Event, error) {
	res, err := s.repository.FindAll(c.GetDatabase(), req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *service) FindByID(c *cctx.Context, ID string) (*models.Event, error) {
	res, err := s.repository.FindByID(c.GetDatabase(), ID)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *service) Count(c *cctx.Context, req *Request) (*int64, error) {
	res, err := s.repository.Count(c.GetDatabase(), req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *service) Insert(c *cctx.Context, req *models.Event) error {
	err := s.repository.Insert(c.GetDatabase(), req)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) SoftDelete(c *cctx.Context, req *models.Event) error {
	err := s.repository.SoftDelete(c.GetDatabase(), req)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) Delete(c *cctx.Context, req *models.Event) error {
	err := s.repository.Delete(c.GetDatabase(), req)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) InsertBlacklist(c *cctx.Context, req *models.Blacklist) error {
	err := s.repository.InsertBlacklist(c.GetDatabase(), req)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) FindBlacklists(c *cctx.Context, req *models.Blacklist) ([]*models.Blacklist, error) {
	res, err := s.repository.FindBlacklists(c.GetDatabase(), req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *service) FindPubkeyBlacklists(c *cctx.Context, req *models.Blacklist) ([]string, error) {
	fetch, err := s.repository.FindBlacklists(c.GetDatabase(), req)
	if err != nil {
		logger.Log.Errorf("find blacklist error: %s", err)
		return nil, err
	}

	res := []string{}
	for _, v := range fetch {
		res = append(res, v.Pubkey)
	}

	return res, nil

}

func (s *service) ClearEventsWithBlacklist(c *cctx.Context) error {
	// find blacklists
	blacklists, err := s.FindPubkeyBlacklists(c, &models.Blacklist{})
	if err != nil {
		logger.Log.Errorf("find blacklist error: %s", err)
		return err
	}

	if generic.IsEmpty(blacklists) {
		return nil
	}

	// find event
	fetch, err := s.FindAll(c, &Request{NostrFilter: &models.Filter{Authors: blacklists}, NoLimit: true})
	if err != nil {
		logger.Log.Errorf("find event with blacklist error: %s", err)
		return err
	}

	// delete event
	for _, v := range fetch {
		err := s.repository.SoftDelete(c.GetDatabase(), &models.Event{ID: v.ID})
		if err != nil {
			logger.Log.Errorf("soft delete event with blacklist error: %s", err)
			return err
		}
	}

	return nil
}

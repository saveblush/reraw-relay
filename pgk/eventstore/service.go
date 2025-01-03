package eventstore

import (
	"github.com/jinzhu/copier"
	"github.com/nbd-wtf/go-nostr"

	"github.com/saveblush/reraw-relay/core/cctx"
	"github.com/saveblush/reraw-relay/core/config"
	"github.com/saveblush/reraw-relay/core/generic"
	"github.com/saveblush/reraw-relay/core/utils/logger"
	"github.com/saveblush/reraw-relay/models"
)

// Service service interface
type Service interface {
	FindAll(c *cctx.Context, req *Request) ([]*nostr.Event, error)
	FindByID(c *cctx.Context, ID string) (*nostr.Event, error)
	Count(c *cctx.Context, req *Request) (*int64, error)
	Insert(c *cctx.Context, req *models.RelayEvent) error
	SoftDelete(c *cctx.Context, req *models.RelayEvent) error
	Delete(c *cctx.Context, req *models.RelayEvent) error
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

func (s *service) FindAll(c *cctx.Context, req *Request) ([]*nostr.Event, error) {
	res, err := s.repository.FindAll(c.GetRelayDatabase(), req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *service) FindByID(c *cctx.Context, ID string) (*nostr.Event, error) {
	fetch, err := s.repository.FindByID(c.GetRelayDatabase(), ID)
	if err != nil {
		return nil, err
	}

	res := &nostr.Event{}
	if !generic.IsEmpty(fetch) {
		copier.Copy(res, fetch)
	}

	return res, nil
}

func (s *service) Count(c *cctx.Context, req *Request) (*int64, error) {
	count, err := s.repository.Count(c.GetRelayDatabase(), req)
	if err != nil {
		return nil, err
	}

	return count, nil
}

func (s *service) Insert(c *cctx.Context, req *models.RelayEvent) error {
	err := s.repository.Insert(c.GetRelayDatabase(), req)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) SoftDelete(c *cctx.Context, req *models.RelayEvent) error {
	err := s.repository.SoftDelete(c.GetRelayDatabase(), req)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) Delete(c *cctx.Context, req *models.RelayEvent) error {
	err := s.repository.Delete(c.GetRelayDatabase(), req)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) InsertBlacklist(c *cctx.Context, req *models.Blacklist) error {
	err := s.repository.InsertBlacklist(c.GetRelayDatabase(), req)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) FindBlacklists(c *cctx.Context, req *models.Blacklist) ([]*models.Blacklist, error) {
	res, err := s.repository.FindBlacklists(c.GetRelayDatabase(), req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *service) FindPubkeyBlacklists(c *cctx.Context, req *models.Blacklist) ([]string, error) {
	res := []string{}
	fetch, err := s.repository.FindBlacklists(c.GetRelayDatabase(), req)
	if err != nil {
		logger.Log.Errorf("find blacklist error: %s", err)
		return nil, err
	}

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
	fetch, err := s.FindAll(c, &Request{NostrFilter: &nostr.Filter{Authors: blacklists}, NoLimit: true})
	if err != nil {
		logger.Log.Errorf("find event with blacklist error: %s", err)
		return err
	}

	// delete event
	for _, v := range fetch {
		err := s.repository.SoftDelete(c.GetRelayDatabase(), &models.RelayEvent{ID: v.ID})
		if err != nil {
			logger.Log.Errorf("soft delete event with blacklist error: %s", err)
			return err
		}
	}

	return nil
}

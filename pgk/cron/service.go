package cron

import (
	"github.com/robfig/cron/v3"

	"github.com/saveblush/reraw-relay/core/cctx"
	"github.com/saveblush/reraw-relay/core/config"
	"github.com/saveblush/reraw-relay/core/utils/logger"
	"github.com/saveblush/reraw-relay/pgk/eventstore"
)

// Service service interface
type Service interface {
	Start()
	Stop()
}

type service struct {
	cctx       *cctx.Context
	config     *config.Configs
	cron       *cron.Cron
	eventstore eventstore.Service
}

func NewService() Service {
	return &service{
		cctx:       cctx.New(),
		config:     config.CF,
		cron:       cron.New(),
		eventstore: eventstore.NewService(),
	}
}

func (s *service) Start() {
	logger.Log.Info("Cron init...")
	s.schedule()
	s.cron.Start()
}

func (s *service) Stop() {
	s.cron.Stop()
}

func (s *service) schedule() {
	// รันทุก 5 นาที
	s.cron.AddFunc("*/5 * * * *", func() {
		s.eventstore.ClearEventsExpiration(s.cctx)
	})

	// รันทุก 30 นาที
	s.cron.AddFunc("*/30 * * * *", func() {
		s.eventstore.ClearEventsWithBlacklist(s.cctx)
	})
}

package policies

import (
	"net/http"

	"github.com/nbd-wtf/go-nostr"

	"github.com/saveblush/reraw-relay/core/cctx"
	"github.com/saveblush/reraw-relay/core/config"
	"github.com/saveblush/reraw-relay/core/generic"
	"github.com/saveblush/reraw-relay/pgk/eventstore"
	"github.com/saveblush/reraw-relay/pgk/nips/nip13"
)

// Service service interface
type Service interface {
	RejectEmptyHeaderUserAgent(r *http.Request) bool
	RejectEmptyFilters(filter *nostr.Filter) (reject bool, msg string)
	RejectEventWithCharacter(c *cctx.Context, evt *nostr.Event) (bool, string)
	RejectValidateEvent(c *cctx.Context, evt *nostr.Event) (bool, string)
	RejectValidatePow(c *cctx.Context, evt *nostr.Event) (bool, string)
	RejectValidateTimeStamp(c *cctx.Context, evt *nostr.Event) (bool, string)
	RejectEventFromPubkeyWithBlacklist(c *cctx.Context, evt *nostr.Event) (bool, string)
	StoreBlacklistWithContent(c *cctx.Context, evt *nostr.Event) error
}

type service struct {
	config     *config.Configs
	eventstore eventstore.Service
	nip13      nip13.Service
}

func NewService() Service {
	return &service{
		config:     config.CF,
		eventstore: eventstore.NewService(),
		nip13:      nip13.NewService(),
	}
}

// RejectEmptyHeaderUserAgent reject empty header user-agent
func (s *service) RejectEmptyHeaderUserAgent(r *http.Request) bool {
	return generic.IsEmpty(r.Header.Get("User-Agent"))
}

// RejectEmptyFilters reject empty filters
func (s *service) RejectEmptyFilters(filter *nostr.Filter) (reject bool, msg string) {
	var c int
	if len(filter.IDs) > 0 {
		c++
	}

	if len(filter.Kinds) > 0 {
		c++
	}

	if len(filter.Authors) > 0 {
		c++
	}

	if len(filter.Tags) > 0 {
		c++
	}

	if filter.Search != "" {
		c++
	}

	if !generic.IsEmpty(filter.Since) {
		c++
	}

	if !generic.IsEmpty(filter.Limit) {
		c++
	}

	if c == 0 {
		return true, nostr.NormalizeOKMessage("can't handle empty filters", "blocked")
	}

	return false, ""
}

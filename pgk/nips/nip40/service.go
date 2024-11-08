package nip40

import (
	"errors"
	"strconv"

	"github.com/nbd-wtf/go-nostr"

	"github.com/saveblush/reraw-relay/core/cctx"
	"github.com/saveblush/reraw-relay/core/config"
)

// Service service interface
type Service interface {
	Expiration(c *cctx.Context, evt *nostr.Event) (int, error)
}

type service struct {
	config *config.Configs
}

func NewService() Service {
	return &service{
		config: config.CF,
	}
}

// Expiration expiration
func (s *service) Expiration(c *cctx.Context, evt *nostr.Event) (int, error) {
	var expiration int
	expirationTag := evt.Tags.GetFirst([]string{"expiration", ""})
	if expirationTag != nil && len(*expirationTag) >= 2 {
		expiration, _ = strconv.Atoi((*expirationTag)[1])
		if expiration < 100 {
			return 0, errors.New("invalid: expiration")
		}
	}

	return expiration, nil
}

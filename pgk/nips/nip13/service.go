package nip13

import (
	"fmt"
	"strconv"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip13"

	"github.com/saveblush/reraw-relay/core/cctx"
	"github.com/saveblush/reraw-relay/core/config"
)

// Service service interface
type Service interface {
	VerifyPow(c *cctx.Context, evt *nostr.Event) (bool, error)
}

type service struct {
	config *config.Configs
}

func NewService() Service {
	return &service{
		config: config.CF,
	}
}

// VerifyPow verify proof of work
func (s *service) VerifyPow(c *cctx.Context, evt *nostr.Event) (bool, error) {
	work := nip13.Difficulty(evt.ID)
	nonceTag := evt.Tags.GetFirst([]string{"nonce", ""})
	if nonceTag != nil && len(*nonceTag) >= 3 {
		target, _ := strconv.Atoi((*nonceTag)[2])
		if work < target {
			return false, fmt.Errorf("difficulty %d is less than %d", work, target)
		}
	}

	if work < s.config.Info.Limitation.MinPowDifficulty {
		return false, fmt.Errorf("insufficient difficulty")
	}

	return true, nil
}

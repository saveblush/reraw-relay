package nip13

import (
	"errors"
	"fmt"
	"math/bits"
	"strconv"

	"github.com/saveblush/reraw-relay/core/cctx"
	"github.com/saveblush/reraw-relay/core/config"
	"github.com/saveblush/reraw-relay/models"
)

// Service service interface
type Service interface {
	VerifyPow(c *cctx.Context, evt *models.Event) (bool, error)
}

type service struct {
	config *config.Configs
}

func NewService() Service {
	return &service{
		config: config.CF,
	}
}

func (s *service) difficulty(hex string) int {
	var count int
	for i := 0; i < len(hex); i++ {
		nibble := int(hex[i] - '0')
		if nibble >= 10 {
			nibble = int(hex[i]-'a') + 10
		}
		if nibble >= 16 {
			nibble = int(hex[i]-'A') + 10
		}
		if nibble == 0 {
			count += 4
		} else {
			count += bits.LeadingZeros32(uint32(nibble)) - 28
			break
		}
	}

	return count
}

// VerifyPow verify proof of work
func (s *service) VerifyPow(c *cctx.Context, evt *models.Event) (bool, error) {
	work := s.difficulty(evt.ID)
	nonceTag := evt.Tags.FindFirst("nonce")
	if nonceTag != nil && len(*nonceTag) >= 3 {
		target, _ := strconv.Atoi((*nonceTag)[2])
		if work < target {
			return false, fmt.Errorf("difficulty %d is less than %d", work, target)
		}
	}

	if work < s.config.Info.Limitation.MinPowDifficulty {
		return false, errors.New("insufficient difficulty")
	}

	return true, nil
}

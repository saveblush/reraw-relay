package policies

import (
	"fmt"
	"strings"

	"github.com/saveblush/reraw-relay/core/cctx"
	"github.com/saveblush/reraw-relay/core/generic"
	"github.com/saveblush/reraw-relay/core/utils/logger"
	"github.com/saveblush/reraw-relay/models"
)

// RejectValidateEvent reject validate event data
func (s *service) RejectValidateEvent(c *cctx.Context, evt *models.Event) (bool, string) {
	if evt.GetID() != evt.ID {
		return true, fmt.Sprintf("invalid: %s", "event id is computed incorrectly")
	}

	ok, err := evt.VerifySignature()
	if err != nil {
		return true, fmt.Sprintf("error: %s", "failed to verify signature")
	}
	if !ok {
		return true, fmt.Sprintf("invalid: %s", "signature is invalid")
	}

	return false, ""
}

// RejectValidatePow reject validate pow
func (s *service) RejectValidatePow(c *cctx.Context, evt *models.Event) (bool, string) {
	pow, err := s.nip13.VerifyPow(c, evt)
	if !pow && err != nil {
		return true, fmt.Sprintf("pow: %s", err)
	}

	return false, ""
}

// RejectValidateTimeStamp reject validate time stamp
func (s *service) RejectValidateTimeStamp(c *cctx.Context, evt *models.Event) (bool, string) {
	if evt.CreatedAt > models.MaxUint32 || evt.Kind > models.MaxUint16 {
		return true, fmt.Sprintf("invalid: %s", "format created_at error")
	}

	return false, ""
}

// RejectEventWithCharacter reject event with character
func (s *service) RejectEventWithCharacter(c *cctx.Context, evt *models.Event) (bool, string) {
	characters := []string{
		//"data:image",
		//"data:video",
	}

	for _, character := range characters {
		if strings.Contains(evt.Content, character) {
			return true, fmt.Sprintf("blocked: event with %s", character)
		}
	}

	return false, ""
}

// RejectEventFromPubkeyWithBlacklist reject event from pubkey with blacklist
func (s *service) RejectEventFromPubkeyWithBlacklist(c *cctx.Context, evt *models.Event) (bool, string) {
	bots, err := s.eventstore.FindBlacklists(c, &models.Blacklist{Pubkey: evt.Pubkey})
	if err != nil {
		logger.Log.Warnf("reject find bot error: %s", err)
	}

	if !generic.IsEmpty(bots) {
		logger.Log.Warnf("found bot: %s", evt.Pubkey)
		return true, fmt.Sprintf("blocked: %s", "hmm.")
	}

	return false, ""
}

// StoreBlacklistWithContent store blacklist with content
func (s *service) StoreBlacklistWithContent(c *cctx.Context, evt *models.Event) error {
	characters := []string{
		"ReplyGuy",
		"ReplyGirl",
	}

	for _, character := range characters {
		if strings.Contains(evt.Content, character) {
			err := s.eventstore.InsertBlacklist(c, &models.Blacklist{Pubkey: evt.Pubkey})
			if err != nil {
				logger.Log.Errorf("keep bot error: %s", err)
				return err
			}
		}
	}

	return nil
}

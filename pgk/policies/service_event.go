package policies

import (
	"fmt"
	"strings"

	"github.com/nbd-wtf/go-nostr"

	"github.com/saveblush/reraw-relay/core/cctx"
	"github.com/saveblush/reraw-relay/core/generic"
	"github.com/saveblush/reraw-relay/core/utils/logger"
	"github.com/saveblush/reraw-relay/models"
)

// RejectValidateEvent reject validate event
func (s *service) RejectValidateEvent(c *cctx.Context, evt *nostr.Event) (bool, string) {
	if evt.GetID() != evt.ID {
		return true, nostr.NormalizeOKMessage("event id is computed incorrectly", "invalid")
	}

	ok, err := evt.CheckSignature()
	if err != nil {
		return true, nostr.NormalizeOKMessage("failed to verify signature", "error")
	} else if !ok {
		return true, nostr.NormalizeOKMessage("signature is invalid", "invalid")
	}

	return false, ""
}

// RejectValidatePow reject validate pow
func (s *service) RejectValidatePow(c *cctx.Context, evt *nostr.Event) (bool, string) {
	pow, err := s.nip13.VerifyPow(c, evt)
	if !pow && err != nil {
		return true, nostr.NormalizeOKMessage(err.Error(), "pow")
	}

	return false, ""
}

// RejectValidateTimeStamp reject validate time stamp
func (s *service) RejectValidateTimeStamp(c *cctx.Context, evt *nostr.Event) (bool, string) {
	if evt.CreatedAt > models.MaxUint32 || evt.Kind > models.MaxUint16 {
		return true, nostr.NormalizeOKMessage("format created_at error", "invalid")
	}

	return false, ""
}

// RejectEventWithCharacter reject event with character
func (s *service) RejectEventWithCharacter(c *cctx.Context, evt *nostr.Event) (bool, string) {
	characters := []string{
		//"data:image",
		//"data:video",
	}

	for _, character := range characters {
		if strings.Contains(evt.Content, character) {
			return true, nostr.NormalizeOKMessage(fmt.Sprintf("event with %s", character), "blocked")
		}
	}

	return false, ""
}

// RejectEventFromPubkeyWithBlacklist reject event from pubkey with blacklist
func (s *service) RejectEventFromPubkeyWithBlacklist(c *cctx.Context, evt *nostr.Event) (bool, string) {
	bots, err := s.eventstore.FindBlacklists(c, &models.Blacklist{Pubkey: evt.PubKey})
	if err != nil {
		logger.Log.Warnf("reject find bot error: %s", err)
	}

	if !generic.IsEmpty(bots) {
		logger.Log.Warnf("found bot: %s", evt.PubKey)
		return true, nostr.NormalizeOKMessage("hmm.", "blocked")
	}

	return false, ""
}

// StoreBlacklistWithContent store blacklist with content
func (s *service) StoreBlacklistWithContent(c *cctx.Context, evt *nostr.Event) error {
	characters := []string{
		"ReplyGuy",
		"ReplyGirl",
	}

	for _, character := range characters {
		if strings.Contains(evt.Content, character) {
			err := s.eventstore.InsertBlacklist(c, &models.Blacklist{Pubkey: evt.PubKey})
			if err != nil {
				logger.Log.Errorf("keep bot error: %s", err)
				return err
			}
		}
	}

	return nil
}

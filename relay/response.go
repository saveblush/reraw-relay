package relay

import (
	"github.com/coder/websocket/wsjson"
	"github.com/nbd-wtf/go-nostr"

	"github.com/saveblush/reraw-relay/core/utils/logger"
)

// websocket response
func (s *service) response(v interface{}) error {
	s.muRes.Lock()
	defer s.muRes.Unlock()

	err := wsjson.Write(s.ctx, s.Conn.Conn, v)
	if err != nil {
		logger.Log.Errorf("write msg error: %s", err)
		return err
	}
	//logger.Log.Info("response msg: ", v)

	return err
}

func (s *service) responseEvent(subID string, evt *nostr.Event) error {
	err := s.response(&nostr.EventEnvelope{SubscriptionID: &subID, Event: *evt})
	if err != nil {
		return err
	}

	return nil
}

func (s *service) responseOK(eventID string, isSuccess bool, reason string) error {
	err := s.response(&nostr.OKEnvelope{EventID: eventID, OK: isSuccess, Reason: reason})
	if err != nil {
		return err
	}

	return nil
}

func (s *service) responseCount(subID string, count *int64) error {
	err := s.response(&nostr.CountEnvelope{SubscriptionID: subID, Count: count})
	if err != nil {
		return err
	}

	return nil
}

// return ปิดการเชื่อมต่อ
func (s *service) responseClosed(subID, reason string) error {
	err := s.response(&nostr.ClosedEnvelope{SubscriptionID: subID, Reason: reason})
	if err != nil {
		return err
	}

	return nil
}

// return เมื่อสิ้นสุดการ REQ
func (s *service) responseEose(subID string) error {
	err := s.response(nostr.EOSEEnvelope(subID))
	if err != nil {
		return err
	}

	return nil
}

func (s *service) responseError(message string) error {
	err := s.response(nostr.NoticeEnvelope(message))
	if err != nil {
		return err
	}

	return nil
}

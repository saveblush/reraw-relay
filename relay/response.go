package relay

import (
	"github.com/saveblush/reraw-relay/models"
)

// websocket response
func (s *service) response(msg interface{}) error {
	/*s.respMutex.Lock()
	defer s.respMutex.Unlock()

	b, err := json.Marshal(&msg)
	if err != nil {
		logger.Log.Errorf("write msg error: %s", err)
		return err
	}

	return s.Conn.WriteMessage(websocket.TextMessage, b)*/

	/*s.respMutex.Lock()
	defer s.respMutex.Unlock()

	b, err := json.Marshal(&msg)
	if err != nil {
		logger.Log.Errorf("write msg error: %s", err)
		return err
	}*/

	//s.Client.send <- msg

	s.client.SendMessage(msg)

	return nil
}

func (s *service) responseEvent(subID string, evt *models.Event) error {
	err := s.response([]interface{}{"EVENT", subID, &evt})
	if err != nil {
		return err
	}

	return nil
}

func (s *service) responseOK(eventID string, isSuccess bool, message string) error {
	err := s.response([]interface{}{"OK", eventID, isSuccess, message})
	if err != nil {
		return err
	}

	return nil
}

func (s *service) responseCount(subID string, count *int64) error {
	err := s.response([]interface{}{"COUNT", subID, count})
	if err != nil {
		return err
	}

	return nil
}

// return ปิดการเชื่อมต่อ
func (s *service) responseClosed(subID, message string) error {
	err := s.response([]interface{}{"CLOSED", subID, message})
	if err != nil {
		return err
	}

	return nil
}

// return เมื่อสิ้นสุดการ REQ
func (s *service) responseEose(subID string) error {
	err := s.response([]interface{}{"EOSE", subID})
	if err != nil {
		return err
	}

	return nil
}

func (s *service) responseError(message string) error {
	err := s.response([]interface{}{"NOTICE", message})
	if err != nil {
		return err
	}

	return nil
}

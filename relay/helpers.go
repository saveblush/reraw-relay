package relay

import (
	"github.com/goccy/go-json"

	"github.com/saveblush/reraw-relay/models"
)

func (s *service) isReplaceableKind(kind int) bool {
	return kind == 0 || kind == 3 || (kind >= 10000 && kind < 20000)
}

func (s *service) isParamReplaceableKind(kind int) bool {
	return kind >= 30000 && kind < 40000
}

func (s *service) isEphemeralKind(kind int) bool {
	return kind >= 20000 && kind < 30000
}

func (s *service) isOlder(previous, next *models.Event) bool {
	return previous.CreatedAt < next.CreatedAt ||
		(previous.CreatedAt == next.CreatedAt && previous.ID > next.ID)
}

func (s *service) subID(req []*json.RawMessage) (string, error) {
	var subID string
	err := json.Unmarshal(*req[1], &subID)
	if err != nil {
		return "", errGetSubID
	}

	return subID, nil
}

func (s *service) filters(req []*json.RawMessage) (*models.Filters, error) {
	filters := make(models.Filters, len(req)-2)
	for i, filterReq := range req[2:] {
		err := json.Unmarshal(*filterReq, &filters[i])
		if err != nil {
			return nil, errInvalidFilter
		}
	}

	return &filters, nil
}

func (s *service) event(req []*json.RawMessage) (*models.Event, error) {
	latestIndex := len(req) - 1
	var evt models.Event
	err := json.Unmarshal(*req[latestIndex], &evt)
	if err != nil {
		return nil, errInvalidEvent
	}

	return &evt, nil
}

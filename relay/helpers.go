package relay

import (
	"github.com/goccy/go-json"
	"github.com/nbd-wtf/go-nostr"

	"github.com/saveblush/reraw-relay/core/generic"
)

func (s *service) isReplaceableKind(kind int) bool {
	return kind == 0 || kind == 3 || kind == 41 || (kind >= 10000 && kind < 20000)
}

func (s *service) isParamReplaceableKind(kind int) bool {
	return kind >= 30000 && kind < 40000
}

func (s *service) isEphemeralKind(kind int) bool {
	return kind >= 20000 && kind < 30000
}

func (s *service) isOlder(previous, next *nostr.Event) bool {
	return previous.CreatedAt < next.CreatedAt ||
		(previous.CreatedAt == next.CreatedAt && previous.ID > next.ID)
}

func (s *service) subID(env interface{}) (string, error) {
	var d []json.RawMessage
	generic.ConvertInterfaceToStruct(env, &d)

	var subID string
	err := json.Unmarshal(d[1], &subID)
	if err != nil {
		return "", errGetSubID
	}

	return subID, nil
}

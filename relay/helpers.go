package relay

import (
	"fmt"

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
	var d []interface{}
	generic.ConvertInterfaceToStruct(env, &d)

	if len(d) < 2 {
		return "", errInvalidClose
	}

	if generic.IsEmpty(d[1]) {
		return "", errSubIDNotFound
	}

	return fmt.Sprintf("%s", d[1]), nil
}

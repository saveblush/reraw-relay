package relay

import (
	"github.com/goccy/go-json"

	"github.com/saveblush/reraw-relay/core/generic"
	"github.com/saveblush/reraw-relay/core/utils"
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
	var id string
	err := json.Unmarshal(*req[1], &id)
	if err != nil {
		return "", errGetSubID
	}

	return id, nil
}

// parseFilters parse filters
func (s *service) parseFilters(req []*json.RawMessage) (*models.Filters, error) {
	if len(req) < 3 {
		return nil, errInvalidFilter
	}

	filters := make(models.Filters, len(req[2:]))
	for i, filter := range req[2:] {
		data := make(map[string]interface{})
		err := json.Unmarshal(*filter, &data)
		if err != nil {
			return nil, errInvalidFilter
		}

		tagMap := make(models.TagMap, 0)
		var out models.Filter
		for k, v := range data {
			switch k {
			case "ids":
				out.IDs = generic.ConvertInterfaceToSliceString(v)

			case "kinds":
				out.Kinds = generic.ConvertInterfaceToSliceInt(v)

			case "authors":
				out.Authors = generic.ConvertInterfaceToSliceString(v)

			case "since":
				out.Since = utils.Pointer(models.Timestamp(generic.ConvertInterfaceToTime(v).Unix()))

			case "until":
				out.Until = utils.Pointer(models.Timestamp(generic.ConvertInterfaceToTime(v).Unix()))

			case "limit":
				out.Limit = generic.ConvertInterfaceToInt(v)

			case "search":
				out.Search = generic.ConvertInterfaceToString(v)

			default:
				if len(k) > 1 && k[0] == '#' {
					tagMap[k] = generic.ConvertInterfaceToSliceString(v)
				}
			}
		}
		out.Tags = tagMap

		filters[i] = out
	}

	return &filters, nil
}

// parseEvent parse event
func (s *service) parseEvent(req []*json.RawMessage) (*models.Event, error) {
	latestIndex := len(req) - 1
	var evt models.Event
	err := json.Unmarshal(*req[latestIndex], &evt)
	if err != nil {
		return nil, errInvalidEvent
	}

	return &evt, nil
}

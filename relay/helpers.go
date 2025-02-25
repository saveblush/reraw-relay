package relay

import (
	"time"

	"github.com/goccy/go-json"
	"github.com/tidwall/gjson"

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
	var subID string
	err := json.Unmarshal(*req[1], &subID)
	if err != nil {
		return "", errGetSubID
	}

	return subID, nil
}

func (s *service) setIntArray(i interface{}) []int {
	if i == nil {
		return nil
	}

	b, err := json.Marshal(i)
	if err != nil {
		return nil
	}
	arr := gjson.ParseBytes(b).Array()

	var result []int
	for _, v := range arr {
		num := v.Num
		result = append(result, int(num))
	}

	return result
}

func (s *service) setStringArray(i interface{}) []string {
	if i == nil {
		return nil
	}

	b, err := json.Marshal(i)
	if err != nil {
		return nil
	}
	arr := gjson.ParseBytes(b).Array()

	var result []string
	for _, v := range arr {
		str := v.Str
		if str != "" {
			result = append(result, str)
		}
	}

	return result
}

func (s *service) setString(i interface{}) string {
	if i == nil {
		return ""
	}

	str, ok := i.(string)
	if !ok {
		return ""
	}

	return str
}

func (s *service) setInt(i interface{}) int {
	if i == nil {
		return 0
	}

	num, ok := i.(float64)
	if !ok {
		return 0
	}

	return int(num)
}

func (s *service) setTime(i interface{}) *models.Timestamp {
	if i == nil {
		return nil
	}

	timestamp, ok := i.(float64)
	if !ok {
		return nil
	}
	t := time.Unix(int64(timestamp), 0).UTC()

	return utils.Pointer(models.Timestamp(t.Unix()))
}

func (s *service) setTagMap(i interface{}) *models.TagMap {
	if i == nil {
		return nil
	}

	tags, ok := i.(map[string]interface{})
	if !ok {
		return nil
	}

	result := make(models.TagMap)
	for k, v := range tags {
		if len(k) > 1 && k[0] == '#' {
			result[k] = s.setStringArray(v)
		}
	}

	return &result
}

// filters Parse and validate filters
func (s *service) filters(req []*json.RawMessage) (*models.Filters, error) {
	filters := make(models.Filters, len(req)-2)
	for i, filter := range req[2:] {
		data := make(map[string]interface{})
		err := json.Unmarshal(*filter, &data)
		if err != nil {
			return nil, errInvalidFilter
		}

		var out models.Filter
		out.IDs = s.setStringArray(data["ids"])
		out.Kinds = s.setIntArray(data["kinds"])
		out.Authors = s.setStringArray(data["authors"])
		out.Tags = *s.setTagMap(data)
		out.Since = s.setTime(data["since"])
		out.Until = s.setTime(data["until"])
		out.Limit = s.setInt(data["limit"])
		out.Search = s.setString(data["search"])

		filters[i] = out
	}

	return &filters, nil
}

// event Parse event
func (s *service) event(req []*json.RawMessage) (*models.Event, error) {
	latestIndex := len(req) - 1
	var evt models.Event
	err := json.Unmarshal(*req[latestIndex], &evt)
	if err != nil {
		return nil, errInvalidEvent
	}

	return &evt, nil
}

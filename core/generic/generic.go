package generic

import (
	"reflect"

	"github.com/goccy/go-json"
)

// IsEmpty is empty
func IsEmpty(i interface{}) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)

	switch v.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		return v.Len() == 0

	case reflect.Ptr:
		if v.IsNil() {
			return true
		}
		ref := v.Elem().Interface()
		return IsEmpty(ref)

	default:
		zero := reflect.Zero(v.Type())
		return reflect.DeepEqual(i, zero.Interface())
	}
}

// ConvertInterfaceToStruct convert interface to struct
func ConvertInterfaceToStruct(data, value interface{}) error {
	b, err := json.Marshal(&data)
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, &value)
	if err != nil {
		return err
	}

	return nil
}

// ConvertEmptyToNull convert empty to null
// แปลงค่าว่างเป็น null เพื่อใช้ยิง db
func ConvertEmptyToNull[T comparable](v T) any {
	if IsEmpty(v) {
		return nil
	}

	return v
}

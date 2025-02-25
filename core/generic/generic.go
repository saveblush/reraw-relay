package generic

import (
	"reflect"
	"time"

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

// ConvertInterfaceToSliceString convert interface to slice string
func ConvertInterfaceToSliceString(i interface{}) []string {
	if i == nil {
		return nil
	}

	arr, ok := i.([]interface{})
	if !ok {
		return nil
	}

	var result []string
	for _, v := range arr {
		str, ok := v.(string)
		if ok {
			result = append(result, str)
		}
	}

	return result
}

// ConvertInterfaceToSliceString convert interface to slice int
func ConvertInterfaceToSliceInt(i interface{}) []int {
	if i == nil {
		return nil
	}

	arr, ok := i.([]interface{})
	if !ok {
		return nil
	}

	var result []int
	for _, v := range arr {
		num, ok := v.(float64)
		if ok {
			result = append(result, int(num))
		}
	}

	return result
}

// ConvertInterfaceToSliceString convert interface to string
func ConvertInterfaceToString(i interface{}) string {
	if i == nil {
		return ""
	}

	result, ok := i.(string)
	if !ok {
		return ""
	}

	return result
}

// ConvertInterfaceToSliceString convert interface to int
func ConvertInterfaceToInt(i interface{}) int {
	if i == nil {
		return 0
	}

	result, ok := i.(float64)
	if !ok {
		return 0
	}

	return int(result)
}

// ConvertInterfaceToTime convert interface to time
func ConvertInterfaceToTime(i interface{}) *time.Time {
	if i == nil {
		return nil
	}

	result, ok := i.(float64)
	if !ok {
		return nil
	}
	t := time.Unix(int64(result), 0).UTC()

	return &t
}

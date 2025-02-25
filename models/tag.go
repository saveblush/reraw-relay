package models

import (
	"errors"
	"fmt"

	"github.com/goccy/go-json"
)

type Tag []string

func (t *Tag) CheckKey(prefix string) bool {
	for i := 0; i < len(*t)-1; i++ {
		if prefix == (*t)[i] {
			return true
		}
	}

	return false
}

func (t *Tag) Key() string {
	if len(*t) > 0 {
		return (*t)[0]
	}

	return ""
}

func (t *Tag) Value() string {
	if len(*t) > 1 {
		return (*t)[1]
	}

	return ""
}

type Tags []Tag

// Scan scan value into Jsonb, implements sql.Scanner interface
func (t *Tags) Scan(v interface{}) error {
	bytes, ok := v.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", v))
	}
	err := json.Unmarshal(bytes, &t)

	return err
}

func (t *Tags) FindKeyD() string {
	for _, v := range *t {
		if v.CheckKey("d") {
			return v[1]
		}
	}

	return ""
}

func (t *Tags) FindFirst(tagPrefix string) *Tag {
	for _, v := range *t {
		if v.CheckKey(tagPrefix) {
			return &v
		}
	}

	return nil
}

func (t *Tags) FindAll(tagPrefix string) *Tags {
	result := make(Tags, 0, len(*t))
	for _, v := range *t {
		if v.CheckKey(tagPrefix) {
			result = append(result, v)
		}
	}

	return &result
}

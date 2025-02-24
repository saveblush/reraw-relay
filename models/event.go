package models

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/goccy/go-json"
)

const (
	MaxUint16 = 65535
	MaxUint32 = 4294967295
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

type Timestamp int64

func (t Timestamp) Time() time.Time {
	return time.Unix(int64(t), 0)
}

type TagMap map[string][]string

type Filter struct {
	IDs     []string
	Kinds   []int
	Authors []string
	Tags    TagMap
	Since   *Timestamp
	Until   *Timestamp
	Limit   int
	Search  string
}

type Filters []Filter

type Subscription struct {
	ID      string
	Filters []Filter
}

type RelayInformationDocument struct {
	Name          string                   `json:"name"`
	Description   string                   `json:"description"`
	Pubkey        string                   `json:"pubkey"`
	Contact       string                   `json:"contact"`
	SupportedNIPs []int                    `json:"supported_nips"`
	Software      string                   `json:"software"`
	Version       string                   `json:"version"`
	Limitation    *RelayLimitationDocument `json:"limitation,omitempty"`
	Icon          string                   `json:"icon"`
}

type RelayLimitationDocument struct {
	MaxMessageLength int  `json:"max_message_length,omitempty"`
	MaxSubscriptions int  `json:"max_subscriptions,omitempty"`
	MaxFilters       int  `json:"max_filters,omitempty"`
	MaxLimit         int  `json:"max_limit,omitempty"`
	MaxSubidLength   int  `json:"max_subid_length,omitempty"`
	MaxEventTags     int  `json:"max_event_tags,omitempty"`
	MaxContentLength int  `json:"max_content_length,omitempty"`
	MinPowDifficulty int  `json:"min_pow_difficulty,omitempty"`
	AuthRequired     bool `json:"auth_required"`
	PaymentRequired  bool `json:"payment_required"`
	RestrictedWrites bool `json:"restricted_writes"`
}

type Event struct {
	ID         string     `json:"id" gorm:"primaryKey;type:varchar(64)"`
	CreatedAt  Timestamp  `json:"created_at" gorm:"type:integer"`
	Pubkey     string     `json:"pubkey" gorm:"type:varchar(64)"`
	Kind       int        `json:"kind" gorm:"type:integer"`
	Content    string     `json:"content"`
	Tags       Tags       `json:"tags" gorm:"type:jsonb"`
	Sig        string     `json:"sig"`
	Tagvalues  []string   `json:"-" gorm:"-"`
	Expiration *Timestamp `json:"-" gorm:"type:integer"`
	UpdatedIP  *string    `json:"-"`
	UpdatedAt  *Timestamp `json:"-" gorm:"type:integer"`
	DeletedAt  *Timestamp `json:"-" gorm:"type:integer"`
}

func (Event) TableName() string {
	return "events"
}

type Blacklist struct {
	gorm.Model
	Pubkey string `json:"pubkey" gorm:"type:varchar(64)"`
}

func (Blacklist) TableName() string {
	return "blacklists"
}

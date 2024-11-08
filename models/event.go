package models

import (
	"gorm.io/gorm"

	"github.com/nbd-wtf/go-nostr"
)

const (
	MaxUint16 = 65535
	MaxUint32 = 4294967295
)

type RelayEvent struct {
	ID         string          `json:"id" gorm:"primaryKey;type:varchar(64)"`
	CreatedAt  nostr.Timestamp `json:"created_at" gorm:"type:integer"`
	UpdatedAt  nostr.Timestamp `json:"-" gorm:"type:integer"`
	DeletedAt  nostr.Timestamp `json:"-" gorm:"type:integer"`
	Pubkey     string          `json:"pubkey" gorm:"type:varchar(64)"`
	Kind       int             `json:"kind" gorm:"type:integer"`
	Content    string          `json:"content"`
	Tags       nostr.Tags      `json:"tags" gorm:"type:jsonb"`
	Sig        string          `json:"sig"`
	Tagvalues  []string        `json:"-" gorm:"-"`
	Expiration nostr.Timestamp `json:"-" gorm:"type:integer"`
	UpdatedIP  string          `json:"-" gorm:"-"`
}

func (RelayEvent) TableName() string {
	return "events"
}

type EventAddon struct {
	UpdatedAt  nostr.Timestamp `json:"updated_at"`
	DeletedAt  nostr.Timestamp `json:"deleted_at"`
	Expiration nostr.Timestamp `json:"expiration"`
	UpdatedIP  string          `json:"updated_ip"`
}

type Blacklist struct {
	gorm.Model
	Pubkey string `json:"pubkey" gorm:"type:varchar(64)"`
}

func (Blacklist) TableName() string {
	return "blacklists"
}

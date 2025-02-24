package models

import (
	"gorm.io/gorm"
)

const (
	MaxUint16 = 65535
	MaxUint32 = 4294967295
)

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

func (evt *Event) CheckSignature() bool {

	return false
}

type Blacklist struct {
	gorm.Model
	Pubkey string `json:"pubkey" gorm:"type:varchar(64)"`
}

func (Blacklist) TableName() string {
	return "blacklists"
}

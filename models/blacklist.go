package models

import "gorm.io/gorm"

type Blacklist struct {
	gorm.Model
	Pubkey string `json:"pubkey" gorm:"type:varchar(64)"`
}

func (Blacklist) TableName() string {
	return "blacklists"
}

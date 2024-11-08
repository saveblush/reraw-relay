package cctx

import (
	"gorm.io/gorm"

	"github.com/saveblush/reraw-relay/core/sql"
)

// GetRelayDatabase get connection database `ralay`
func (c *Context) GetRelayDatabase() *gorm.DB {
	return sql.RelayDatabase
}

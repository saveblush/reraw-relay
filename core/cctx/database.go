package cctx

import (
	"gorm.io/gorm"

	"github.com/saveblush/reraw-relay/core/sql"
)

// GetDatabase get connection database `ralay`
func (c *Context) GetDatabase() *gorm.DB {
	return sql.Database
}

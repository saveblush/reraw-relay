package sql

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/saveblush/reraw-relay/core/utils"
)

// openPostgres open initialize a new db connection.
func openPostgres(cf *Configuration) (*gorm.DB, error) {
	dsn := fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=%s TimeZone=%s sslmode=disable",
		cf.Username,
		cf.Password,
		cf.Host,
		cf.Port,
		cf.DatabaseName,
		utils.TimeZone(),
	)

	return gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // disables implicit prepared statement usage
	}), defaultConfig)
}

package sql

import (
	"errors"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/saveblush/reraw-relay/core/generic"
	"github.com/saveblush/reraw-relay/core/utils"
	"github.com/saveblush/reraw-relay/core/utils/logger"
	"github.com/saveblush/reraw-relay/models"
)

func createDatabase(cf *Configuration) error {
	dsn := fmt.Sprintf("user=%s password=%s host=%s port=%d sslmode=disable TimeZone=%s",
		cf.Username,
		cf.Password,
		cf.Host,
		cf.Port,
		utils.TimeZone(),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	var exc string
	sql := "SELECT 'CREATE DATABASE " + cf.DatabaseName + "' WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = ?)"
	err = db.Raw(sql, cf.DatabaseName).Scan(&exc).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Log.Errorf("check already database error: %s", err)
	}
	if !generic.IsEmpty(exc) {
		err := db.Exec(exc).Error
		if err != nil {
			logger.Log.Errorf("create database error: %s", err)
			return err
		}
	}

	return nil
}

func Migration(db *gorm.DB) error {
	var sqls []string
	sqls = append(sqls, `
		CREATE OR REPLACE FUNCTION json_value_to_array(jsonb)
			RETURNS text[]
			LANGUAGE SQL
			IMMUTABLE
		AS 'SELECT array_agg(t->>1) FROM (SELECT jsonb_array_elements($1) AS t)s WHERE length(t->>0) = 1;'
		RETURNS NULL ON NULL INPUT;
	`)

	sqls = append(sqls, `
		CREATE TABLE IF NOT EXISTS events (
			id varchar(64) NOT NULL PRIMARY KEY,
			created_at integer DEFAULT NULL,
			updated_at integer DEFAULT NULL,
			deleted_at integer DEFAULT NULL,
			pubkey varchar(64) DEFAULT NULL,
			kind integer DEFAULT NULL,
			tags jsonb DEFAULT NULL,
			content text DEFAULT NULL,
			sig text DEFAULT NULL,
			tagvalues text[] GENERATED ALWAYS AS (json_value_to_array(tags)) STORED,
			expiration integer DEFAULT NULL
 		);
	`)

	// index events
	sqls = append(sqls, `CREATE INDEX IF NOT EXISTS idx_events_id ON events (id);`)
	sqls = append(sqls, `CREATE INDEX IF NOT EXISTS idx_events_pubkey ON events (pubkey);`)
	sqls = append(sqls, `CREATE INDEX IF NOT EXISTS idx_events_created_at ON events (created_at DESC);`)
	sqls = append(sqls, `CREATE INDEX IF NOT EXISTS idx_events_deleted_at ON events (deleted_at);`)
	sqls = append(sqls, `CREATE INDEX IF NOT EXISTS idx_events_kind ON events (kind);`)
	sqls = append(sqls, `CREATE INDEX IF NOT EXISTS idx_events_tagvalues ON events USING gin (tagvalues);`)
	sqls = append(sqls, `CREATE INDEX IF NOT EXISTS idx_events_expiration ON events (expiration);`)

	for _, sql := range sqls {
		err := db.Exec(sql).Error
		if err != nil {
			logger.Log.Errorf("db migration error: %s", err)
			return err
		}
	}

	db.AutoMigrate(&models.Blacklist{})

	return nil
}

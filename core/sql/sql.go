package sql

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	// RelayDatabase Database global variable database `relay`
	RelayDatabase = &gorm.DB{}
)

var (
	defaultMaxIdleConns = 20
	defaultMaxOpenConns = 30
	defaultMaxLifetime  = time.Minute
)

// gorm config
var defaultConfig = &gorm.Config{
	PrepareStmt:          true,
	DisableAutomaticPing: true,
	Logger:               logger.Default.LogMode(logger.Error),
}

// Session session
type Session struct {
	Database *gorm.DB
	Conn     *sql.DB
}

// Configuration config mysql
type Configuration struct {
	Host         string
	Port         int
	Username     string
	Password     string
	DatabaseName string
	MaxIdleConns int
	MaxOpenConns int
	MaxLifetime  time.Duration
}

// InitConnectionMysql open initialize a new db connection.
func InitConnection(cf *Configuration) (*Session, error) {
	var db *gorm.DB
	var err error

	// create database
	err = createDatabase(cf)
	if err != nil {
		return nil, err
	}

	// connect db postgres
	db, err = openPostgres(cf)
	if err != nil {
		return nil, err
	}

	// set config connection pool
	if cf.MaxIdleConns > 0 {
		cf.MaxIdleConns = defaultMaxIdleConns
	}
	if cf.MaxOpenConns > 0 {
		cf.MaxOpenConns = defaultMaxOpenConns
	}
	if cf.MaxLifetime > 0 {
		cf.MaxLifetime = defaultMaxLifetime
	}

	// connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(cf.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cf.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cf.MaxLifetime)

	err = sqlDB.Ping()
	if err != nil {
		return nil, err
	}

	return &Session{Database: db, Conn: sqlDB}, nil
}

// CloseConnection close connection db
func CloseConnection(db *gorm.DB) error {
	c, err := db.DB()
	if err != nil {
		return err
	}

	err = c.Close()
	if err != nil {
		return err
	}

	return nil
}

// DebugRelayDatabase set debug sql
func DebugRelayDatabase() {
	RelayDatabase = RelayDatabase.Debug()
}

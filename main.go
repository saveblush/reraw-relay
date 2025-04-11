package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/saveblush/reraw-relay/core/config"
	"github.com/saveblush/reraw-relay/core/sql"
	"github.com/saveblush/reraw-relay/core/utils/logger"
	"github.com/saveblush/reraw-relay/pgk/cron"
	"github.com/saveblush/reraw-relay/relay"
)

func main() {
	flag.Parse()

	// Init logger
	logger.InitLogger()

	// Init configuration
	err := config.InitConfig()
	if err != nil {
		logger.Log.Panicf("init configuration error: %s", err)
	}

	// Init connection database
	cfdb := &sql.Configuration{
		Host:         config.CF.Database.RelaySQL.Host,
		Port:         config.CF.Database.RelaySQL.Port,
		Username:     config.CF.Database.RelaySQL.Username,
		Password:     config.CF.Database.RelaySQL.Password,
		DatabaseName: config.CF.Database.RelaySQL.DatabaseName,
		MaxIdleConns: config.CF.Database.RelaySQL.MaxIdleConns,
		MaxOpenConns: config.CF.Database.RelaySQL.MaxOpenConns,
		MaxLifetime:  config.CF.Database.RelaySQL.MaxLifetime,
	}
	session, err := sql.InitConnection(cfdb)
	if err != nil {
		logger.Log.Panicf("init connection db error: %s", err)
	}

	// Set to global variable database
	sql.Database = session.Database

	// Debug db
	if !config.CF.App.Environment.Production() {
		sql.DebugDatabase()
	}

	// Migration db
	_ = sql.Migration(sql.Database)

	// Cron
	cron := cron.NewService()
	cron.Start()

	// Init relay
	rl := relay.NewRelay()
	handler := rl.Serve()

	// Start app
	addr := flag.String("addr", fmt.Sprintf(":%d", config.CF.App.Port), "http service address")
	server := &http.Server{
		Addr:    *addr,
		Handler: handler,
	}
	server.SetKeepAlivesEnabled(true)

	go func() {
		err = server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Panicf("App start error: %s", err)
		}
	}()
	logger.Log.Infof("App start on: %s", *addr)

	// Shutdown app
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Close relay
	go rl.CloseRelay()
	logger.Log.Info("Relay closed")

	// Close cron
	go cron.Stop()
	logger.Log.Info("Cron closed")

	// Close db
	go sql.CloseConnection(sql.Database)
	logger.Log.Info("Database connection closed")

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownRelease()

	err = server.Shutdown(shutdownCtx)
	if err != nil {
		logger.Log.Panicf("App shutdown error: %s", err)
	}
	logger.Log.Info("Gracefully shutting down")
}

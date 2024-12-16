package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jinzhu/copier"
	"github.com/lesismal/nbio/nbhttp"
	"github.com/nbd-wtf/go-nostr/nip11"
	"github.com/rs/cors"

	"github.com/saveblush/reraw-relay/core/config"
	"github.com/saveblush/reraw-relay/core/sql"
	"github.com/saveblush/reraw-relay/core/utils/logger"
	"github.com/saveblush/reraw-relay/pgk/cron"
	"github.com/saveblush/reraw-relay/relay"
)

const (
	// MaximumSize body limit
	MaximumSize1MB = 1024 * 1024 * 1

	// Timeout
	Timeout75s = time.Second * 75
	Timeout60s = time.Second * 60
	Timeout45s = time.Second * 45
	Timeout30s = time.Second * 30
	Timeout5s  = time.Second * 5
)

func main() {
	flag.Parse()

	// Init logger
	logger.InitLogger()

	// Init configuration
	err := config.InitConfig()
	if err != nil {
		logger.Log.Fatalf("init configuration error: %s", err)
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
		logger.Log.Fatalf("init connection db error: %s", err)
	}

	// Set to global variable database
	sql.RelayDatabase = session.Database

	// Debug db
	if !config.CF.App.Environment.Production() {
		sql.DebugRelayDatabase()
	}

	// Migration db
	_ = sql.Migration(sql.RelayDatabase)

	// Init relay
	nip11 := &nip11.RelayInformationDocument{}
	copier.Copy(nip11, &config.CF.Info)
	rl := relay.NewRelay(&relay.Relay{
		Info:               nip11,
		KeepaliveTime:      Timeout75s,
		HandshakeTimeout:   Timeout45s,
		MessageLengthLimit: MaximumSize1MB,
	})

	// Init server
	mux := &http.ServeMux{}
	mux.HandleFunc("/", rl.HandleWebsocket)

	// Init app
	engine := nbhttp.NewEngine(nbhttp.Config{
		Name:                    config.CF.Info.Name,
		Network:                 "tcp",
		Addrs:                   []string{fmt.Sprintf(":%d", config.CF.App.Port)},
		Handler:                 cors.Default().Handler(mux),
		ReleaseWebsocketPayload: true,
		ReadBufferSize:          1024 * 16,
		ReadLimit:               MaximumSize1MB,
		MaxHTTPBodySize:         MaximumSize1MB,
		WriteTimeout:            Timeout30s,
		IOMod:                   nbhttp.IOModBlocking,
	})

	// Start app
	err = engine.Start()
	if err != nil {
		logger.Log.Fatalf("app start error: %s", err)
		return
	}

	// Cron
	cron := cron.NewService()
	cron.Start()

	// Shutdown app
	exit, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	serverShutdown := make(chan struct{})
	go func() {
		<-exit.Done()
		logger.Log.Info("Gracefully shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), Timeout5s)
		defer cancel()

		// Shutdown engine
		engine.Shutdown(ctx)
		serverShutdown <- struct{}{}
	}()

	// Cleanup tasks
	<-serverShutdown
	logger.Log.Info("Running cleanup tasks...")

	// Close relay
	go rl.CloseRelay()
	logger.Log.Info("Relay closed")

	// Close cron
	go cron.Stop()
	logger.Log.Info("Cron closed")

	// Close db
	go sql.CloseConnection(sql.RelayDatabase)
	logger.Log.Info("Database connection closed")

	logger.Log.Info("App was successful shutdown")
}

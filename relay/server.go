package relay

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/idempotency"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/pprof"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"

	"github.com/saveblush/reraw-relay/core/config"
	"github.com/saveblush/reraw-relay/core/utils"
)

// NewServer new server
func NewServer() *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:           config.CF.Info.Name,
		ServerHeader:      config.CF.Info.Name,
		BodyLimit:         MaximumSize1MB,
		ReadBufferSize:    MaximumSize1MB,
		WriteBufferSize:   MaximumSize1MB,
		IdleTimeout:       Timeout15s,
		ReadTimeout:       Timeout10s,
		WriteTimeout:      Timeout5s,
		ReduceMemoryUsage: true,
		CaseSensitive:     true,
		JSONEncoder:       utils.Marshal,
		JSONDecoder:       utils.Unmarshal,
	})

	// Middlewares
	app.Use(
		compress.New(compress.Config{
			Level: compress.LevelBestCompression,
		}),
		cors.New(),
		requestid.New(),
		idempotency.New(),
		pprof.New(),
		recover.New(),
	)

	// Logging
	app.Use(logger.New(logger.Config{
		Format: "[${ip}]:${port} ${status} - ${method} ${path}\n",
	}))

	// Limiter
	app.Use(limiter.New(limiter.Config{
		Max:        221,
		Expiration: time.Minute * 1,
	}))

	return app
}

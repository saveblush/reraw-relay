package relay

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip11"
	"github.com/saveblush/gofiber3-contrib/websocket"

	"github.com/saveblush/reraw-relay/core/cctx"
	"github.com/saveblush/reraw-relay/core/utils/logger"
	"github.com/saveblush/reraw-relay/pgk/policies"
)

var (
	errConnectDatabase = errors.New("error: could not connect to the database")
	errInvalidMessage  = errors.New("error: invalid message")
	errDuplicate       = errors.New("duplicate: already have this event")
	errInvalidClose    = errors.New("error: invalid CLOSE")
	errUnknownCommand  = errors.New("error: unknown command")
	errSubIDNotFound   = errors.New("error: subscription id not found")
)

var (
	pingInterval = time.Second * 15
	pingWait     = time.Second * 20
)

type StoreEvent []func(cctx *cctx.Context, evt *nostr.Event) error
type RejectConnection []func(r *http.Request) bool
type RejectFilter []func(filter *nostr.Filter) (reject bool, msg string)
type RejectEvent []func(cctx *cctx.Context, evt *nostr.Event) (reject bool, msg string)

type Relay struct {
	Info *nip11.RelayInformationDocument

	KeepaliveTime      time.Duration
	HandshakeTimeout   time.Duration
	MessageLengthLimit int

	policies policies.Service

	clients      map[*websocket.Conn][]string
	clientsMutex sync.Mutex
}

// NewRelay new relay
func NewRelay(rl *Relay) (app *fiber.App, relay *Relay) {
	srv := NewServer()
	srv.Get("/", rl.HandleWebsocket())

	rl.clients = make(map[*websocket.Conn][]string)
	rl.policies = policies.NewService()

	return srv, rl
}

const (
	// MaximumSize10MB body limit 1 mb.
	MaximumSize10MB = 10 * 1024 * 1024
	// MaximumSize1MB body limit 1 mb.
	MaximumSize1MB = 1 * 1024 * 1024
	// Timeout timeout 15 seconds
	Timeout15s = 15 * time.Second
	// Timeout timeout 10 seconds
	Timeout10s = 10 * time.Second
	// Timeout timeout 5 seconds
	Timeout5s = 5 * time.Second
)

// HandleWebsocket handle websocket
func (rl *Relay) HandleWebsocket() fiber.Handler {
	return websocket.New(func(conn *websocket.Conn) {
		conn.EnableWriteCompression(true)
		conn.SetCompressionLevel(2)
		conn.SetReadLimit(int64(rl.MessageLengthLimit))
		conn.SetReadDeadline(time.Now().Add(time.Second * 20))

		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(
					err,
					websocket.CloseNormalClosure,    // 1000
					websocket.CloseGoingAway,        // 1001
					websocket.CloseNoStatusReceived, // 1005
					websocket.CloseAbnormalClosure,  // 1006
				) {
					logger.Log().Errorf("read msg error: %s", err)
				}
				break
			}

			if mt != websocket.TextMessage {
				logger.Log().Errorf("%s is sending non-UTF-8 data. disconnecting....", conn.IP)
				_ = conn.Close()
				return
			} else if mt == websocket.PingMessage {
				_ = conn.WriteMessage(websocket.PongMessage, nil)
				continue
			}

			// event nostr
			storeEvent := append(StoreEvent{}, rl.policies.StoreBlacklistWithContent)
			rejectFilter := append(RejectFilter{}, rl.policies.RejectEmptyFilters)
			rejectEvent := append(RejectEvent{},
				rl.policies.RejectValidateEvent,
				rl.policies.RejectValidatePow,
				rl.policies.RejectValidateTimeStamp,
				rl.policies.RejectEventFromPubkeyWithBlacklist,
				rl.policies.RejectEventWithCharacter)

			sess := &session{
				Ws:           conn,
				StoreEvent:   storeEvent,
				RejectFilter: rejectFilter,
				RejectEvent:  rejectEvent,
			}
			rt := newHandleEvent(sess)
			err = rt.handleEvent(msg)
			if err != nil {
				logger.Log().Errorf("handle event error: %s", err)
				return
			}
		}
	}, websocket.Config{
		HandshakeTimeout:  rl.HandshakeTimeout,
		EnableCompression: true,
	})
}

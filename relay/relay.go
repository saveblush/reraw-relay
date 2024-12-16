package relay

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/lesismal/nbio/nbhttp/websocket"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip11"

	"github.com/saveblush/reraw-relay/core/cctx"
	"github.com/saveblush/reraw-relay/core/utils"
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
func NewRelay(rl *Relay) *Relay {
	rl.clients = make(map[*websocket.Conn][]string)
	rl.policies = policies.NewService()

	return rl
}

// CloseRelay close relay
func (rl *Relay) CloseRelay() error {
	rl.clientsMutex.Lock()
	defer rl.clientsMutex.Unlock()

	for c := range rl.clients {
		c.WriteClose(1000, "normal close")
		c.Close()
	}
	clear(rl.clients)

	return nil
}

// HandleWebsocket handle websocket
func (rl *Relay) HandleWebsocket(w http.ResponseWriter, r *http.Request) {
	// check reject
	rejectConnection := append(RejectConnection{}, rl.policies.RejectEmptyHeaderUserAgent)
	for _, rejectFunc := range rejectConnection {
		if rejectFunc(r) {
			return
		}
	}

	if r.Method == http.MethodGet && r.Header.Get("Upgrade") == "websocket" {
		rl.handleMessage(w, r)
	} else {
		if len(r.Header.Get("Upgrade")) > 0 {
			http.Error(w, "Invalid Upgrade Header", http.StatusBadRequest)
			return
		}

		if strings.Contains(r.Header.Get("Accept"), "application/nostr+json") {
			rl.showNIP11(w)
		} else {
			rl.showInfo(w)
		}
	}
}

// newUpgrader new upgrader
func (rl *Relay) newUpgrader() *websocket.Upgrader {
	upgrader := websocket.NewUpgrader()
	upgrader.EnableCompression(true)
	upgrader.SetCompressionLevel(8)
	upgrader.BlockingModAsyncWrite = true
	upgrader.BlockingModHandleRead = false
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	if rl.HandshakeTimeout > 0 {
		upgrader.HandshakeTimeout = rl.HandshakeTimeout
	}
	if rl.KeepaliveTime > 0 {
		upgrader.KeepaliveTime = rl.KeepaliveTime
	}
	if rl.MessageLengthLimit > 0 {
		upgrader.MessageLengthLimit = rl.MessageLengthLimit
	}

	upgrader.OnOpen(func(c *websocket.Conn) {
		logger.Log.Info("onOpen: ", c.RemoteAddr().String())

		_ = c.SetDeadline(time.Now().Add(pingInterval + pingWait))

		rl.clientsMutex.Lock()
		rl.clients[c] = make([]string, 0, 2)
		rl.clientsMutex.Unlock()
	})

	upgrader.OnClose(func(c *websocket.Conn, err error) {
		logger.Log.Info("onClose: ", c.RemoteAddr().String(), err)

		rl.clientsMutex.Lock()
		delete(rl.clients, c)
		rl.clientsMutex.Unlock()
	})

	return upgrader
}

// HandleMessage handle message
func (rl *Relay) handleMessage(w http.ResponseWriter, r *http.Request) {
	upgrader := rl.newUpgrader()
	up, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Log.Panicf("upgrader error: %s", err)
	}

	// set event reject
	up.OnMessage(func(c *websocket.Conn, mt websocket.MessageType, msg []byte) {
		c.HandleRead(1024 * 16)

		if mt != websocket.TextMessage {
			logger.Log.Error("message is not UTF-8. disconnecting...")
			_ = c.Close()
			return
		} else if mt == websocket.PingMessage {
			logger.Log.Warn("ping... pong...")
			_ = c.SetDeadline(time.Now().Add(pingInterval + pingWait))
			_ = c.WriteMessage(websocket.PongMessage, nil)
			return
		}

		// event nostr
		storeEvent := append(StoreEvent{}, rl.policies.StoreBlacklistWithContent)
		rejectFilter := append(RejectFilter{}, rl.policies.RejectEmptyFilters)
		rejectEvent := append(RejectEvent{},
			rl.policies.RejectValidateEvent,
			rl.policies.RejectValidatePow,
			rl.policies.RejectValidateTimeStamp,
			rl.policies.RejectEventWithCharacter,
			rl.policies.RejectEventFromPubkeyWithBlacklist)

		sess := &session{
			Ws:           c,
			StoreEvent:   storeEvent,
			RejectFilter: rejectFilter,
			RejectEvent:  rejectEvent,
		}
		rt := newHandleEvent(sess)
		err := rt.handleEvent(msg)
		if err != nil {
			logger.Log.Errorf("handle event error: %s", err)
			return
		}
		c.SetReadDeadline(time.Now().Add(time.Second * 30))
	})
}

// showNIP11 show nip11 info
func (rl *Relay) showNIP11(w http.ResponseWriter) {
	b, err := utils.Marshal(rl.Info)
	if err != nil {
		fmt.Fprintf(w, "{}")
		return
	}

	_, _ = w.Write(b)
	w.Header().Set("Content-Type", "application/nostr+json")
}

// showInfo show html info
func (rl *Relay) showInfo(w http.ResponseWriter) {
	supportedNIPs := rl.Info.SupportedNIPs
	arrSupportedNIPs := make([]string, len(supportedNIPs))
	for i, v := range supportedNIPs {
		arrSupportedNIPs[i] = fmt.Sprintf("%v", v)
	}

	var str []string
	str = append(str, fmt.Sprintf("Name: %s", rl.Info.Name))
	str = append(str, fmt.Sprintf("Description: %s", rl.Info.Description))
	str = append(str, fmt.Sprintf("PubKey: %s", rl.Info.PubKey))
	str = append(str, fmt.Sprintf("Contact: %s", rl.Info.Contact))
	str = append(str, fmt.Sprintf("SupportedNIPs: %s", strings.Join(arrSupportedNIPs, ", ")))
	str = append(str, fmt.Sprintf("Version: %s", rl.Info.Version))
	fmt.Fprint(w, strings.Join(str, "\n"))
}

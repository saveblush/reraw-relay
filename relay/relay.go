package relay

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"
	"github.com/jinzhu/copier"

	"github.com/saveblush/reraw-relay/core/cctx"
	"github.com/saveblush/reraw-relay/core/config"
	"github.com/saveblush/reraw-relay/core/generic"
	"github.com/saveblush/reraw-relay/core/utils/logger"
	"github.com/saveblush/reraw-relay/models"
	"github.com/saveblush/reraw-relay/pgk/policies"
)

const (
	pongWait = 120 * time.Second
)

var (
	errConnectDatabase      = errors.New("error: could not connect to the database")
	errInvalidMessage       = errors.New("error: invalid message")
	errInvalidParamsMessage = errors.New("error: request has less than 2 parameters")
	errInvalidReq           = errors.New("error: invalid REQ")
	errInvalidFilter        = errors.New("error: failed to decode filter")
	errInvalidEvent         = errors.New("error: failed to decode event")
	errDuplicate            = errors.New("duplicate: already have this event")
	errUnknownCommand       = errors.New("error: unknown command")
	errSubIDNotFound        = errors.New("error: subscription id not found")
	errGetSubID             = errors.New("error: received subscription id is not a string")
)

type StoreEvent []func(cctx *cctx.Context, evt *models.Event) error
type RejectConnection []func(r *http.Request) bool
type RejectFilter []func(filter *models.Filter) (reject bool, msg string)
type RejectEvent []func(cctx *cctx.Context, evt *models.Event) (reject bool, msg string)

type Relay struct {
	serveMux *http.ServeMux
	ctx      context.Context
	upgrader websocket.Upgrader
	policies policies.Service

	ServiceURL string
	Info       *models.RelayInformationDocument

	HandshakeTimeout   time.Duration
	MessageLengthLimit int64

	clientsMutex sync.Mutex
	clients      map[*websocket.Conn]struct{}
}

// NewRelay new relay
func NewRelay() *Relay {
	nip11 := &models.RelayInformationDocument{}
	copier.Copy(nip11, &config.CF.Info)

	rl := &Relay{
		serveMux: &http.ServeMux{},
		ctx:      context.TODO(),
		upgrader: websocket.Upgrader{
			ReadBufferSize:    1024,
			WriteBufferSize:   1024,
			EnableCompression: true,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		policies: policies.NewService(),

		Info:               nip11,
		HandshakeTimeout:   180 * time.Second,
		MessageLengthLimit: 1024 * 1024 * 0.5,

		clients: make(map[*websocket.Conn]struct{}),
	}

	return rl
}

func (rl *Relay) Serve() *http.ServeMux {
	mux := rl.serveMux
	mux.HandleFunc("/", rl.handleRequest)

	return mux
}

// CloseRelay close relay
func (rl *Relay) CloseRelay() error {
	rl.clientsMutex.Lock()
	defer rl.clientsMutex.Unlock()

	for c := range rl.clients {
		c.WriteControl(websocket.CloseMessage, nil, time.Now().Add(time.Second))
		c.Close()
	}
	clear(rl.clients)

	return nil
}

// handleRequest handle request
func (rl *Relay) handleRequest(w http.ResponseWriter, r *http.Request) {
	// check reject
	rejectConnection := append(RejectConnection{}, rl.policies.RejectEmptyHeaderUserAgent)
	for _, rejectFunc := range rejectConnection {
		if rejectFunc(r) {
			return
		}
	}

	if r.Method == http.MethodGet && r.Header.Get("Upgrade") == "websocket" {
		rl.handleWebsocket(w, r)
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

// handleWebsocket handle websocket
func (rl *Relay) handleWebsocket(w http.ResponseWriter, r *http.Request) {
	// policies event nostr
	storeEvent := append(StoreEvent{}, rl.policies.StoreBlacklistWithContent)
	rejectFilter := append(RejectFilter{}, rl.policies.RejectEmptyFilters)
	rejectEvent := append(RejectEvent{},
		rl.policies.RejectValidateEvent,
		rl.policies.RejectValidatePow,
		rl.policies.RejectValidateTimeStamp,
		rl.policies.RejectEventWithCharacter,
		rl.policies.RejectEventFromPubkeyWithBlacklist)

	// ip
	ip := rl.ip(r)

	// ws
	up := rl.upgrader
	up.HandshakeTimeout = rl.HandshakeTimeout
	ws, err := up.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			logger.Log.Error("ws upgrade error: %s", err)
		}
		return
	}
	defer func() {
		rl.clientsMutex.Lock()
		if _, ok := rl.clients[ws]; ok {
			ws.Close()
			delete(rl.clients, ws)
		}
		rl.clientsMutex.Unlock()
		logger.Log.Infof("[disconnect] %s", ip)
	}()

	// config ws
	if config.CF.Info.Limitation.MaxMessageLength > 0 {
		rl.MessageLengthLimit = int64(config.CF.Info.Limitation.MaxMessageLength)
	}
	ws.SetReadLimit(rl.MessageLengthLimit)
	ws.SetCompressionLevel(9)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	conn := &Conn{
		Conn: ws,
		ip:   ip,
	}
	logger.Log.Infof("[connected] %s", ip)

	// clients ws
	rl.clientsMutex.Lock()
	rl.clients[ws] = struct{}{}
	rl.clientsMutex.Unlock()

	// handle event nostr
	rt := newHandleEvent()
	rt.Conn = conn
	rt.StoreEvent = storeEvent
	rt.RejectFilter = rejectFilter
	rt.RejectEvent = rejectEvent

	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseNormalClosure,
				websocket.CloseAbnormalClosure,
				websocket.CloseNoStatusReceived,
				websocket.CloseGoingAway,
			) {
				logger.Log.Warnf("unexpected close error from %s: %s", conn.IP(), err)
			}
			break
		}

		if mt != websocket.TextMessage {
			logger.Log.Error("message is not UTF-8. %s disconnecting...", conn.IP())
			break
		}

		if mt == websocket.PingMessage {
			conn.WriteMessage(websocket.PongMessage, nil)
			continue
		}

		go func(msg []byte) {
			err = rt.handleEvent(msg)
			if err != nil {
				logger.Log.Errorf("handle event error: %s", err)
				return
			}
		}(msg)
	}
}

// ip get the client's ip address
func (rl *Relay) ip(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	ip := strings.Split(xff, ",")[0]
	if generic.IsEmpty(ip) {
		ip = r.RemoteAddr
	}

	return ip
}

// showNIP11 show nip11 info
func (rl *Relay) showNIP11(w http.ResponseWriter) {
	b, err := json.Marshal(&rl.Info)
	if err != nil {
		fmt.Fprintf(w, "{}")
		return
	}

	w.Header().Set("Content-Type", "application/nostr+json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	_, _ = w.Write(b)
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
	str = append(str, fmt.Sprintf("PubKey: %s", rl.Info.Pubkey))
	str = append(str, fmt.Sprintf("Contact: %s", rl.Info.Contact))
	str = append(str, fmt.Sprintf("SupportedNIPs: %s", strings.Join(arrSupportedNIPs, ", ")))
	str = append(str, fmt.Sprintf("Version: %s", rl.Info.Version))

	fmt.Fprint(w, strings.Join(str, "\n"))
}

type Conn struct {
	*websocket.Conn
	ip string
}

// IP returns the client's ip address
func (conn *Conn) IP() string {
	return conn.ip
}

package relay

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"
	"github.com/jinzhu/copier"
	"golang.org/x/time/rate"

	"github.com/saveblush/reraw-relay/core/cctx"
	"github.com/saveblush/reraw-relay/core/config"
	"github.com/saveblush/reraw-relay/core/utils"
	"github.com/saveblush/reraw-relay/core/utils/limiter"
	"github.com/saveblush/reraw-relay/core/utils/logger"
	"github.com/saveblush/reraw-relay/models"
	"github.com/saveblush/reraw-relay/pgk/policies"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

var (
	errConnectDatabase      = errors.New("error: could not connect to the database")
	errInvalidMessage       = errors.New("error: invalid message")
	errInvalidParamsMessage = errors.New("error: request has less than 2 parameters")
	errInvalidFilter        = errors.New("error: failed to decode filter")
	errInvalidEvent         = errors.New("error: failed to decode event")
	errDuplicateEvent       = errors.New("duplicate: already have this event")
	errUnknownCommand       = errors.New("error: unknown command")
	errSubIDNotFound        = errors.New("error: subscription id not found")
	errGetSubID             = errors.New("error: received subscription ID is not a string")
	errrInvalidESubID       = errors.New("invalid: subscription ID must be between 1 and 64 characters")
)

type Relay struct {
	serveMux *http.ServeMux
	mu       sync.Mutex

	policies         policies.Service
	rejectConnection []func(r *http.Request) bool
	storeEvent       []func(cctx *cctx.Context, evt *models.Event) error
	rejectFilter     []func(filter *models.Filter) (reject bool, msg string)
	rejectEvent      []func(cctx *cctx.Context, evt *models.Event) (reject bool, msg string)

	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client

	limiter         *limiter.IPRateLimiter
	limiterBlockIPs map[string]bool

	ServiceURL   string
	Info         *models.RelayInformationDocument
	faviconBytes []byte

	HandshakeTimeout   time.Duration
	WriteWait          time.Duration
	PongWait           time.Duration
	PingPeriod         time.Duration
	MessageLengthLimit int64
}

// NewRelay new relay
func NewRelay() *Relay {
	rl := &Relay{
		serveMux: &http.ServeMux{},
		policies: policies.NewService(),

		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),

		limiterBlockIPs: make(map[string]bool),

		HandshakeTimeout:   360 * time.Second,
		WriteWait:          10 * time.Second,
		PongWait:           180 * time.Second,
		PingPeriod:         90 * time.Second,
		MessageLengthLimit: 0.5 * 1024 * 1024,
	}

	// info relay
	nip11 := &models.RelayInformationDocument{}
	copier.Copy(nip11, &config.CF.Info)
	rl.Info = nip11

	// policies event nostr
	rl.rejectConnection = append(rl.rejectConnection, rl.policies.RejectEmptyHeaderUserAgent)
	rl.storeEvent = append(rl.storeEvent, rl.policies.StoreBlacklistWithContent)
	rl.rejectFilter = append(rl.rejectFilter, rl.policies.RejectEmptyFilters)
	rl.rejectEvent = append(rl.rejectEvent,
		rl.policies.RejectValidateEvent,
		rl.policies.RejectValidatePow,
		rl.policies.RejectValidateTimeStamp,
		rl.policies.RejectEventWithCharacter,
		rl.policies.RejectEventFromPubkeyWithBlacklist)

	// retelimit
	if config.CF.App.RateLimit.Enable {
		rl.limiter = limiter.NewIPRateLimiter(rate.Limit(config.CF.App.RateLimit.Limit), config.CF.App.RateLimit.Burst)
	}

	rl.loadFavicon()
	go rl.ready()

	return rl
}

func (rl *Relay) Serve() *http.ServeMux {
	mux := rl.serveMux
	mux.HandleFunc("/favicon.ico", rl.handleFavicon)
	mux.HandleFunc("/", rl.handleRequest)

	return mux
}

func (rl *Relay) ready() {
	for {
		select {
		case client := <-rl.register:
			rl.clients[client] = true
			logger.Log.Infof("[connected] %s", client.IP())

		case client := <-rl.unregister:
			if _, ok := rl.clients[client]; ok {
				delete(rl.clients, client)
				logger.Log.Infof("[disconnect] %s", client.Info())
			}
		}
	}
}

// CloseRelay close relay
func (rl *Relay) CloseRelay() error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	for c := range rl.clients {
		c.conn.WriteControl(websocket.CloseMessage, nil, time.Now().Add(time.Second))
		c.conn.Close()
	}
	clear(rl.clients)

	return nil
}

// handleRequest handle request
func (rl *Relay) handleRequest(w http.ResponseWriter, r *http.Request) {
	// check reject
	for _, rejectFunc := range rl.rejectConnection {
		if rejectFunc(r) {
			http.Error(w, "Invalid Upgrade Header", http.StatusBadRequest)
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
	// limiter block ip
	ip := utils.GetIP(r)
	_, exists := rl.limiterBlockIPs[ip]
	if exists {
		logger.Log.Warnf("limiter block ip: %s", ip)
		return
	}

	// ws
	up := upgrader
	up.HandshakeTimeout = rl.HandshakeTimeout
	conn, err := up.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			logger.Log.Error("websocket upgrade error: %s", err)
		}
		return
	}

	// client
	client := &Client{
		relay:       rl,
		conn:        conn,
		ip:          ip,
		userAgent:   utils.GetUserAgent(r),
		connectedAt: utils.Now(),
	}
	client.relay.register <- client

	// อ่านข้อความ
	client.reader()
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
	str = append(str, fmt.Sprintf("Software: %s", rl.Info.Software))
	str = append(str, fmt.Sprintf("Version: %s", rl.Info.Version))

	_, _ = w.Write([]byte(strings.Join(str, "\n")))
}

// handleFavicon handle favicon
func (rl *Relay) handleFavicon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/x-icon")
	w.Header().Set("Cache-Control", "public, max-age=7776000")

	_, _ = w.Write(rl.faviconBytes)
}

// LoadFavicon load favicon
func (rl *Relay) loadFavicon() {
	if rl.Info.Icon == "" {
		return
	}

	resp, err := http.Get(rl.Info.Icon)
	if err == nil && resp.StatusCode == http.StatusOK {
		var buffer bytes.Buffer
		if _, err = io.Copy(&buffer, resp.Body); err != nil {
			return
		}
		rl.faviconBytes = buffer.Bytes()
	}
}

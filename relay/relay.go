package relay

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"
	"github.com/jinzhu/copier"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip11"

	"github.com/saveblush/reraw-relay/core/cctx"
	"github.com/saveblush/reraw-relay/core/config"
	"github.com/saveblush/reraw-relay/core/generic"
	"github.com/saveblush/reraw-relay/core/utils/logger"
	"github.com/saveblush/reraw-relay/pgk/policies"
)

const (
	defaultMessageLengthLimit = 1024 * 1024 * 0.5

	// Time allowed to read the next pong message from the client.
	pongWait = 60 * time.Second
)

var (
	errConnectDatabase = errors.New("error: could not connect to the database")
	errInvalidMessage  = errors.New("error: invalid message")
	errDuplicate       = errors.New("duplicate: already have this event")
	//errInvalidClose    = errors.New("error: invalid CLOSE")
	errUnknownCommand = errors.New("error: unknown command")
	errSubIDNotFound  = errors.New("error: subscription id not found")
	errGetSubID       = errors.New("error: received subscription id is not a string")
)

type StoreEvent []func(cctx *cctx.Context, evt *nostr.Event) error
type RejectConnection []func(r *http.Request) bool
type RejectFilter []func(filter *nostr.Filter) (reject bool, msg string)
type RejectEvent []func(cctx *cctx.Context, evt *nostr.Event) (reject bool, msg string)

type Relay struct {
	serveMux *http.ServeMux
	ctx      context.Context
	upgrader websocket.Upgrader
	policies policies.Service

	ServiceURL string
	Info       *nip11.RelayInformationDocument

	HandshakeTimeout   time.Duration
	MessageLengthLimit int64
}

// NewRelay new relay
func NewRelay() *Relay {
	nip11 := &nip11.RelayInformationDocument{}
	copier.Copy(nip11, &config.CF.Info)

	rl := &Relay{
		serveMux: &http.ServeMux{},
		ctx:      context.TODO(),
		upgrader: websocket.Upgrader{
			ReadBufferSize:    4096,
			WriteBufferSize:   4096,
			EnableCompression: true,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		policies: policies.NewService(),

		Info:               nip11,
		HandshakeTimeout:   90 * time.Second,
		MessageLengthLimit: int64(config.CF.Info.Limitation.MaxMessageLength),
	}

	return rl
}

func (rl *Relay) Serve() *http.ServeMux {
	mux := rl.serveMux
	mux.HandleFunc("/", rl.handleRequest)

	return mux
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

	if rl.MessageLengthLimit <= 0 {
		rl.MessageLengthLimit = int64(defaultMessageLengthLimit)
	}

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

	conn := &Conn{
		Conn: ws,
		ip:   rl.ip(r),
	}
	logger.Log.Infof("[connected] %s", conn.IP())

	// handle event nostr
	rt := newHandleEvent()
	rt.Conn = conn
	rt.StoreEvent = storeEvent
	rt.RejectFilter = rejectFilter
	rt.RejectEvent = rejectEvent

	go func() {
		defer func() {
			conn.Close()
			logger.Log.Infof("[disconnect] %s", conn.IP())
		}()

		conn.SetReadLimit(rl.MessageLengthLimit)
		conn.SetCompressionLevel(9)
		conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

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
	}()
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
	str = append(str, fmt.Sprintf("PubKey: %s", rl.Info.PubKey))
	str = append(str, fmt.Sprintf("Contact: %s", rl.Info.Contact))
	str = append(str, fmt.Sprintf("SupportedNIPs: %s", strings.Join(arrSupportedNIPs, ", ")))
	str = append(str, fmt.Sprintf("Version: %s", rl.Info.Version))
	fmt.Fprint(w, strings.Join(str, "\n"))
}

type Conn struct {
	*websocket.Conn
	ip string
}

/*var poolConn = sync.Pool{
	New: func() interface{} {
		return new(Conn)
	},
}

// acquireConn acquire conn from pool
func acquireConn() *Conn {
	conn := poolConn.Get().(*Conn)
	return conn
}

// releaseConn return conn to pool
func releaseConn(conn *Conn) {
	conn.Conn = nil
	poolConn.Put(conn)
}*/

// IP returns the client's ip address
func (conn *Conn) IP() string {
	return conn.ip
}

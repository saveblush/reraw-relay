package relay

import (
	"fmt"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"

	"github.com/saveblush/reraw-relay/core/config"
	"github.com/saveblush/reraw-relay/core/utils/logger"
)

type Client struct {
	relay     *Relay
	conn      *websocket.Conn
	send      chan []byte
	respMutex sync.Mutex

	ip          string
	userAgent   string
	connectedAt time.Time
}

func (client *Client) IP() string {
	return client.ip
}

func (client *Client) UserAgent() string {
	return client.userAgent
}

func (client *Client) Info() string {
	return fmt.Sprintf("IP: %s, Connected At: %s", client.ip, client.connectedAt.Format(time.RFC3339))
}

// reader อ่านข้อความจาก client
func (client *Client) reader() {
	defer func() {
		client.relay.unregister <- client
		client.conn.Close()
	}()

	// config การเชื่อมต่อ websocket
	if config.CF.Info.Limitation.MaxMessageLength > 0 {
		client.relay.MessageLengthLimit = int64(config.CF.Info.Limitation.MaxMessageLength)
	}
	client.conn.SetReadLimit(client.relay.MessageLengthLimit)
	client.conn.SetCompressionLevel(9)
	client.conn.SetReadDeadline(time.Now().Add(client.relay.PongWait))
	client.conn.SetPongHandler(func(string) error { client.conn.SetReadDeadline(time.Now().Add(client.relay.PongWait)); return nil })

	rt := newHandleEvent()
	rt.client = client

	for {
		mt, msg, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseNormalClosure,
				websocket.CloseAbnormalClosure,
				websocket.CloseNoStatusReceived,
				websocket.CloseGoingAway,
			) {
				logger.Log.Warnf("unexpected close error from %s: %s", client.IP(), err)
			}
			break
		}

		if mt != websocket.TextMessage {
			logger.Log.Error("message is not UTF-8. %s disconnecting...", client.IP())
			break
		}

		if mt == websocket.PingMessage {
			client.conn.WriteMessage(websocket.PongMessage, nil)
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

// writer ส่งข้อความจาก relay ไปยัง client
func (client *Client) writer() {
	for msg := range client.send {
		err := client.conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			logger.Log.Errorf("write msg error: %s", err)
			return
		}
	}
}

// SendMessage ข้อความจาก relay เตรียมส่งไปยัง client
func (client *Client) SendMessage(msg interface{}) error {
	client.respMutex.Lock()
	defer client.respMutex.Unlock()

	b, err := json.Marshal(&msg)
	if err != nil {
		return err
	}
	client.send <- b

	return nil
}

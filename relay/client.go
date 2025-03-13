package relay

import (
	"time"

	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"

	"github.com/saveblush/reraw-relay/core/config"
	"github.com/saveblush/reraw-relay/core/utils/logger"
)

type Client struct {
	relay *Relay

	conn *websocket.Conn

	send chan []byte

	ip string
}

// IP get client's ip
func (client *Client) IP() string {
	return client.ip
}

// reader pumps messages from the websocket connection to the relay
func (client *Client) reader() {
	defer func() {
		client.relay.unregister <- client
		client.conn.Close()
	}()

	// config conn
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

// writer pumps messages from the relay to the websocket connection
func (client *Client) writer() {
	ticker := time.NewTicker(client.relay.PingPeriod)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(client.relay.WriteWait))
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := client.conn.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				logger.Log.Errorf("write msg error: %s", err)
				return
			}

		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(client.relay.WriteWait))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (client *Client) SendMessage(msg interface{}) error {
	b, err := json.Marshal(&msg)
	if err != nil {
		return err
	}
	client.send <- b

	return nil
}

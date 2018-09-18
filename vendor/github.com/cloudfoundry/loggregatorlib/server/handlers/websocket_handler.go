package handlers

import (
	"net/http"
	"time"

	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/loggregatorlib/server"
	"github.com/gorilla/websocket"
)

type websocketHandler struct {
	messages  <-chan []byte
	keepAlive time.Duration
	logger    *gosteno.Logger
}

func NewWebsocketHandler(m <-chan []byte, keepAlive time.Duration, logger *gosteno.Logger) *websocketHandler {
	return &websocketHandler{messages: m, keepAlive: keepAlive, logger: logger}
}

func (h *websocketHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	h.logger.Debugf("websocket handler: ServeHTTP entered with request %v", r.URL)

	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}

	ws, err := upgrader.Upgrade(rw, r, nil)
	if err != nil {
		h.logger.Errorf("websocket handler: Not a websocket handshake: %s", err.Error())
		return
	}
	defer ws.Close()

	closeCode, closeMessage := h.runWebsocketUntilClosed(ws)
	ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(closeCode, closeMessage), time.Time{})
}

func (h *websocketHandler) runWebsocketUntilClosed(ws *websocket.Conn) (closeCode int, closeMessage string) {
	keepAliveExpired := make(chan struct{})
	clientWentAway := make(chan struct{})

	// TODO: remove this loop (but keep ws.ReadMessage()) once we retire support in the cli for old style keep alives
	go func() {
		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				close(clientWentAway)
				h.logger.Debugf("websocket handler: connection from %s was closed", ws.RemoteAddr().String())
				return
			}
		}
	}()

	go func() {
		server.NewKeepAlive(ws, h.keepAlive).Run()
		close(keepAliveExpired)
		h.logger.Debugf("websocket handler: Connection from %s timed out", ws.RemoteAddr().String())
	}()

	closeCode = websocket.CloseNormalClosure
	closeMessage = ""
	for {
		select {
		case <-clientWentAway:
			return
		case <-keepAliveExpired:
			closeCode = websocket.ClosePolicyViolation
			closeMessage = "Client did not respond to ping before keep-alive timeout expired."
			return
		case message, ok := <-h.messages:
			if !ok {
				h.logger.Debug("websocket handler: messages channel was closed")
				return
			}
			err := ws.WriteMessage(websocket.BinaryMessage, message)
			if err != nil {
				h.logger.Errorf("websocket handler: Error writing to websocket: %s", err.Error())
				return
			}
		}
	}
}

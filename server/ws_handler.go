package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"

	"meow-ai/config"
	"meow-ai/voice"
)

type Handler struct {
	cfg      *config.Config
	upgrader websocket.Upgrader
}

func NewHandler(cfg *config.Config) *Handler {
	return &Handler{
		cfg: cfg,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/ws/realtime", h.handleRealtime)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

type clientStartMessage struct {
	Type       string `json:"type"`
	SampleRate int    `json:"sampleRate"`
	Encoding   string `json:"encoding"`
}

type clientControlMessage struct {
	Type string `json:"type"`
}

func (h *Handler) handleRealtime(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		glog.Errorf("upgrade websocket: %v", err)
		return
	}
	defer conn.Close()

	startMsg, err := h.readStart(conn)
	if err != nil {
		h.writeError(conn, err)
		return
	}

	format := voice.InputFormat{
		SampleRate: startMsg.SampleRate,
		Encoding:   voice.Encoding(startMsg.Encoding),
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	session, err := voice.NewSession(ctx, h.cfg, format)
	if err != nil {
		h.writeError(conn, err)
		return
	}
	defer session.Close()

	writer := &wsWriter{conn: conn}
	if err := writer.writeJSON(map[string]any{"type": "ready"}); err != nil {
		return
	}

	errCh := make(chan error, 2)
	go func() {
		errCh <- h.pipeFrontend(conn, session)
	}()
	go func() {
		errCh <- h.pipeBackend(writer, session)
	}()

	err = <-errCh
	cancel()
	if err != nil && !errors.Is(err, context.Canceled) && !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
		glog.Warningf("ws session ended with error: %v", err)
	}
}

func (h *Handler) readStart(conn *websocket.Conn) (clientStartMessage, error) {
	if err := conn.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
		return clientStartMessage{}, err
	}
	mt, data, err := conn.ReadMessage()
	if err != nil {
		return clientStartMessage{}, err
	}
	if mt != websocket.TextMessage {
		return clientStartMessage{}, errors.New("期待 type=start 的文本消息")
	}
	var msg clientStartMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return clientStartMessage{}, err
	}
	if msg.Type != "start" {
		return clientStartMessage{}, errors.New("首条消息必须是 {type:\"start\"}")
	}
	if msg.SampleRate == 0 {
		msg.SampleRate = 48000
	}
	if msg.Encoding == "" {
		msg.Encoding = string(voice.EncodingF32)
	}
	return msg, nil
}

func (h *Handler) pipeFrontend(conn *websocket.Conn, session *voice.Session) error {
	for {
		if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
			return err
		}
		mt, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		switch mt {
		case websocket.BinaryMessage:
			if err := session.PushAudio(data); err != nil {
				return err
			}
		case websocket.TextMessage:
			var msg clientControlMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			if msg.Type == "stop" {
				return nil
			}
		default:
			glog.Infof("ignore message type=%d", mt)
		}
	}
}

func (h *Handler) pipeBackend(writer *wsWriter, session *voice.Session) error {
	// Handle both audio and events
	for {
		select {
		case data, ok := <-session.Audio():
			if !ok {
				return session.Err() // Channel closed
			}
			if len(data) == 0 {
				continue
			}
			if err := writer.writeBinary(data); err != nil {
				return err
			}
		case evt, ok := <-session.Events():
			if !ok {
				return session.Err()
			}
			// Forward event to frontend
			// Convert payload to RawMessage to avoid double encoding if it is already JSON bytes
			// Actually `evt.Payload` is []byte, which will be base64 encoded if we put it in struct directly as []byte
			// We want it to be a nested JSON object.
			
			jsonMsg := map[string]any{
				"type":     evt.Type,
				"event_id": evt.EventID,
				"payload":  json.RawMessage(evt.Payload),
			}
			
			if err := writer.writeJSON(jsonMsg); err != nil {
				return err
			}
		}
	}
}

func (h *Handler) writeError(conn *websocket.Conn, err error) {
	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_ = conn.WriteJSON(map[string]any{
		"type":    "error",
		"message": err.Error(),
	})
}

type wsWriter struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (w *wsWriter) writeJSON(v any) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}
	return w.conn.WriteJSON(v)
}

func (w *wsWriter) writeBinary(data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}
	return w.conn.WriteMessage(websocket.BinaryMessage, data)
}

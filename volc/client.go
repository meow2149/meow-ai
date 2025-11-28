package volc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"meow-ai/config"
)

const (
	eventStartConnection  int32 = 1
	eventFinishConnection int32 = 2
	eventStartSession     int32 = 100
	eventFinishSession    int32 = 102
	eventSayHello         int32 = 300
	eventUserQuery        int32 = 200
)

var (
	readTimeout  = 0 * time.Second
	writeTimeout = 5 * time.Second
)

type Client struct {
	cfg       *config.Config
	conn      *websocket.Conn
	sessionID string

	jsonProto *BinaryProtocol
	rawProto  *BinaryProtocol

	sendMu sync.Mutex
}

type StartSessionPayload struct {
	ASR    ASRPayload    `json:"asr"`
	TTS    TTSPayload    `json:"tts"`
	Dialog DialogPayload `json:"dialog"`
}

type ASRPayload struct {
	Extra map[string]any `json:"extra"`
}

type TTSPayload struct {
	Speaker     string      `json:"speaker"`
	AudioConfig AudioConfig `json:"audio_config"`
}

type AudioConfig struct {
	Channel    int    `json:"channel"`
	Format     string `json:"format"`
	SampleRate int    `json:"sample_rate"`
}

type DialogPayload struct {
	DialogID          string                 `json:"dialog_id,omitempty"`
	BotName           string                 `json:"bot_name"`
	SystemRole        string                 `json:"system_role"`
	SpeakingStyle     string                 `json:"speaking_style"`
	CharacterManifest string                 `json:"character_manifest,omitempty"`
	Location          *config.LocationConfig `json:"location,omitempty"`
	Extra             map[string]any         `json:"extra"`
}

type SayHelloPayload struct {
	Content string `json:"content"`
}

func NewClient(cfg *config.Config) *Client {
	jsonProto := newBaseProtocol()
	jsonProto.SetSerialization(SerializationJSON)

	rawProto := jsonProto.Clone()
	rawProto.SetSerialization(SerializationRaw)

	return &Client{
		cfg:       cfg,
		jsonProto: jsonProto,
		rawProto:  rawProto,
	}
}

func newBaseProtocol() *BinaryProtocol {
	p := NewBinaryProtocol()
	p.SetVersion(Version1)
	p.SetHeaderSize(HeaderSize4)
	p.SetCompression(CompressionNone, nil)
	p.containsSequence = ContainsSequence
	return p
}

func (c *Client) Open(ctx context.Context) error {
	if c.conn != nil {
		return fmt.Errorf("client already opened")
	}

	dialCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	conn, resp, err := websocket.DefaultDialer.DialContext(dialCtx, c.cfg.API.URL, http.Header{
		"X-Api-Resource-Id": []string{c.cfg.API.ResourceID},
		"X-Api-Access-Key":  []string{c.cfg.API.AccessKey},
		"X-Api-App-Key":     []string{c.cfg.API.AppKey},
		"X-Api-App-ID":      []string{c.cfg.API.AppID},
		"X-Api-Connect-Id":  []string{uuid.NewString()},
	})
	if err != nil {
		return fmt.Errorf("dial doubao api: %w", err)
	}
	if resp != nil {
		glog.Infof("doubao logid: %s", resp.Header.Get("X-Tt-Logid"))
	}

	c.conn = conn
	c.sessionID = uuid.NewString()

	if err := c.startConnection(ctx); err != nil {
		conn.Close()
		return err
	}
	if err := c.startSession(ctx); err != nil {
		conn.Close()
		return err
	}
	return nil
}

func (c *Client) startConnection(ctx context.Context) error {
	msg, err := NewMessage(MsgTypeFullClient, MsgTypeFlagWithEvent)
	if err != nil {
		return fmt.Errorf("new start connection message: %w", err)
	}
	msg.Event = eventStartConnection
	msg.Payload = []byte("{}")

	if err := c.writeMessage(ctx, msg, SerializationJSON); err != nil {
		return fmt.Errorf("send start connection: %w", err)
	}

	resp, err := c.readMessage(ctx)
	if err != nil {
		return fmt.Errorf("wait connection started: %w", err)
	}
	if resp.Type != MsgTypeFullServer || resp.Event != 50 {
		return fmt.Errorf("unexpected connection response: type=%s event=%d", resp.Type, resp.Event)
	}
	glog.Infof("doubao connection established, connect_id=%s", resp.ConnectID)
	return nil
}

func (c *Client) startSession(ctx context.Context) error {
	payload := StartSessionPayload{
		ASR: ASRPayload{
			Extra: map[string]any{
				"end_smooth_window_ms": c.cfg.Session.ASR.Extra.EndSmoothWindowMS,
				"enable_custom_vad":    c.cfg.Session.ASR.Extra.EnableCustomVAD,
				"enable_asr_twopass":   c.cfg.Session.ASR.Extra.EnableASRTwoPass,
			},
		},
		TTS: TTSPayload{
			Speaker: c.cfg.Session.TTS.Speaker,
			AudioConfig: AudioConfig{
				Channel:    c.cfg.Session.TTS.AudioConfig.Channel,
				Format:     c.cfg.Session.TTS.AudioConfig.Format,
				SampleRate: c.cfg.Session.TTS.AudioConfig.SampleRate,
			},
		},
		Dialog: DialogPayload{
			DialogID:          c.cfg.Session.Dialog.DialogID,
			BotName:           c.cfg.Session.Dialog.BotName,
			SystemRole:        c.cfg.Session.Dialog.SystemRole,
			SpeakingStyle:     c.cfg.Session.Dialog.SpeakingStyle,
			CharacterManifest: c.cfg.Session.Dialog.CharacterManifest,
			Location:          c.cfg.Session.Dialog.Location,
			Extra: map[string]any{
				"strict_audit":                     c.cfg.Session.Dialog.Extra.StrictAudit,
				"audit_response":                   c.cfg.Session.Dialog.Extra.AuditResponse,
				"enable_volc_websearch":            c.cfg.Session.Dialog.Extra.EnableVolcWebsearch,
				"volc_websearch_type":              c.cfg.Session.Dialog.Extra.VolcWebsearchType,
				"volc_websearch_api_key":           c.cfg.Session.Dialog.Extra.VolcWebsearchAPIKey,
				"volc_websearch_result_count":      c.cfg.Session.Dialog.Extra.VolcWebsearchResultCount,
				"volc_websearch_no_result_message": c.cfg.Session.Dialog.Extra.VolcWebsearchNoResultMsg,
				"input_mod":                        c.cfg.Session.Dialog.Extra.InputMod,
				"model":                            c.cfg.Session.Dialog.Extra.Model,
				"recv_timeout":                     c.cfg.Session.Dialog.Extra.RecvTimeout,
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal start session payload: %w", err)
	}

	msg, err := NewMessage(MsgTypeFullClient, MsgTypeFlagWithEvent)
	if err != nil {
		return fmt.Errorf("new start session message: %w", err)
	}
	msg.Event = eventStartSession
	msg.SessionID = c.sessionID
	msg.Payload = body

	if err := c.writeMessage(ctx, msg, SerializationJSON); err != nil {
		return fmt.Errorf("send start session: %w", err)
	}
	resp, err := c.readMessage(ctx)
	if err != nil {
		return fmt.Errorf("wait start session response: %w", err)
	}
	if resp.Type != MsgTypeFullServer || resp.Event != 150 {
		return fmt.Errorf("unexpected start session response: type=%s event=%d payload=%s", resp.Type, resp.Event, string(resp.Payload))
	}
	glog.Infof("doubao session started, session_id=%s", resp.SessionID)
	return nil
}

func (c *Client) SayHello(ctx context.Context, content string) error {
	msg, err := NewMessage(MsgTypeFullClient, MsgTypeFlagWithEvent)
	if err != nil {
		return fmt.Errorf("new sayHello message: %w", err)
	}
	msg.Event = eventSayHello
	msg.SessionID = c.sessionID
	body, err := json.Marshal(SayHelloPayload{Content: content})
	if err != nil {
		return fmt.Errorf("marshal sayHello payload: %w", err)
	}
	msg.Payload = body
	return c.writeMessage(ctx, msg, SerializationJSON)
}

func (c *Client) SendAudio(ctx context.Context, pcm []byte) error {
	msg, err := NewMessage(MsgTypeAudioOnlyClient, MsgTypeFlagWithEvent)
	if err != nil {
		return fmt.Errorf("new audio message: %w", err)
	}
	msg.Event = eventUserQuery
	msg.SessionID = c.sessionID
	msg.Payload = pcm
	return c.writeMessage(ctx, msg, SerializationRaw)
}

func (c *Client) readMessage(ctx context.Context) (*Message, error) {
	if ctx != nil {
		if readTimeout > 0 {
			_ = c.conn.SetReadDeadline(time.Now().Add(readTimeout))
		} else {
			_ = c.conn.SetReadDeadline(time.Time{})
		}
	}
	mt, frame, err := c.conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	if mt != websocket.BinaryMessage && mt != websocket.TextMessage {
		return nil, fmt.Errorf("unsupported message type: %d", mt)
	}
	msg, _, err := Unmarshal(frame, ContainsSequence)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (c *Client) Read(ctx context.Context) (*Message, error) {
	return c.readMessage(ctx)
}

func (c *Client) ReadLoop(ctx context.Context, fn func(*Message) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		msg, err := c.readMessage(ctx)
		if err != nil {
			return err
		}
		if err := fn(msg); err != nil {
			return err
		}
	}
}

func (c *Client) writeMessage(ctx context.Context, msg *Message, serialization SerializationBits) error {
	proto := c.jsonProto
	if serialization == SerializationRaw {
		proto = c.rawProto
	}
	frame, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	if ctx != nil {
		_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	}
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	return c.conn.WriteMessage(websocket.BinaryMessage, frame)
}

func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.finishSession(ctx); err != nil {
		glog.Warningf("finish session error: %v", err)
	}
	if err := c.finishConnection(ctx); err != nil {
		glog.Warningf("finish connection error: %v", err)
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

func (c *Client) finishSession(ctx context.Context) error {
	msg, err := NewMessage(MsgTypeFullClient, MsgTypeFlagWithEvent)
	if err != nil {
		return fmt.Errorf("new finish session message: %w", err)
	}
	msg.Event = eventFinishSession
	msg.SessionID = c.sessionID
	msg.Payload = []byte("{}")

	if err := c.writeMessage(ctx, msg, SerializationJSON); err != nil {
		return fmt.Errorf("send finish session: %w", err)
	}
	return nil
}

func (c *Client) finishConnection(ctx context.Context) error {
	msg, err := NewMessage(MsgTypeFullClient, MsgTypeFlagWithEvent)
	if err != nil {
		return fmt.Errorf("new finish connection message: %w", err)
	}
	msg.Event = eventFinishConnection
	msg.Payload = []byte("{}")
	if err := c.writeMessage(ctx, msg, SerializationJSON); err != nil {
		return fmt.Errorf("send finish connection: %w", err)
	}
	resp, err := c.readMessage(ctx)
	if err != nil {
		return fmt.Errorf("wait finish connection response: %w", err)
	}
	if resp.Type != MsgTypeFullServer || resp.Event != 52 {
		return fmt.Errorf("unexpected finish connection response: type=%s event=%d", resp.Type, resp.Event)
	}
	return nil
}

func (c *Client) SessionID() string {
	return c.sessionID
}

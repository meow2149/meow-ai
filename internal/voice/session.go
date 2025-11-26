package voice

import (
	"context"
	"fmt"
	"sync"

	"github.com/golang/glog"

	"meow-ai/internal/config"
	"meow-ai/internal/volc"
)

type Session struct {
	client    *volc.Client
	processor *PCMProcessor

	audioCh chan []byte

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	errMu sync.Mutex
	err   error
}

func NewSession(parent context.Context, cfg *config.Config, format InputFormat) (*Session, error) {
	processor, err := NewPCMProcessor(format)
	if err != nil {
		return nil, err
	}
	client := volc.NewClient(cfg)
	ctx, cancel := context.WithCancel(parent)

	if err := client.Open(ctx); err != nil {
		cancel()
		return nil, fmt.Errorf("open doubao session: %w", err)
	}
	greeting := fmt.Sprintf("你好，我是%s，有什么可以帮助你的吗？", cfg.Session.Dialog.BotName)
	if err := client.SayHello(ctx, greeting); err != nil {
		cancel()
		client.Close()
		return nil, fmt.Errorf("send greeting: %w", err)
	}

	s := &Session{
		client:    client,
		processor: processor,
		audioCh:   make(chan []byte, 64),
		ctx:       ctx,
		cancel:    cancel,
	}

	s.wg.Add(1)
	go s.consume()
	return s, nil
}

func (s *Session) consume() {
	defer s.wg.Done()
	defer close(s.audioCh)

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}
		msg, err := s.client.Read(s.ctx)
		if err != nil {
			s.setError(fmt.Errorf("read from doubao: %w", err))
			return
		}
		switch msg.Type {
		case volc.MsgTypeAudioOnlyServer:
			payload := make([]byte, len(msg.Payload))
			copy(payload, msg.Payload)
			select {
			case s.audioCh <- payload:
			case <-s.ctx.Done():
				return
			}
		case volc.MsgTypeFullServer:
			if msg.Event == 152 || msg.Event == 153 {
				glog.Infof("doubao session closed event=%d", msg.Event)
				return
			}
		case volc.MsgTypeError:
			s.setError(fmt.Errorf("doubao error code=%d payload=%s", msg.ErrorCode, string(msg.Payload)))
			return
		default:
			glog.Infof("ignore doubao message type=%s event=%d", msg.Type, msg.Event)
		}
	}
}

func (s *Session) setError(err error) {
	if err == nil {
		return
	}
	s.errMu.Lock()
	defer s.errMu.Unlock()
	if s.err == nil {
		s.err = err
		s.cancel()
	}
}

func (s *Session) Audio() <-chan []byte {
	return s.audioCh
}

func (s *Session) PushAudio(frame []byte) error {
	if len(frame) == 0 {
		return nil
	}
	select {
	case <-s.ctx.Done():
		return s.Err()
	default:
	}
	pcm, err := s.processor.Process(frame)
	if err != nil {
		return err
	}
	if len(pcm) == 0 {
		return nil
	}
	return s.client.SendAudio(s.ctx, pcm)
}

func (s *Session) Close() error {
	s.cancel()
	s.wg.Wait()
	return s.client.Close()
}

func (s *Session) Err() error {
	s.errMu.Lock()
	defer s.errMu.Unlock()
	return s.err
}

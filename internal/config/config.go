package config

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  ServerConfig  `yaml:"server"`
	API     APIConfig     `yaml:"api"`
	Session SessionConfig `yaml:"session"`
}

type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

type APIConfig struct {
	URL        string `yaml:"url"`
	AppID      string `yaml:"app_id"`
	AppKey     string `yaml:"app_key"`
	ResourceID string `yaml:"resource_id"`
	AccessKey  string `yaml:"access_key"`
}

type SessionConfig struct {
	ASR    ASRConfig    `yaml:"asr"`
	TTS    TTSConfig    `yaml:"tts"`
	Dialog DialogConfig `yaml:"dialog"`
}

type ASRConfig struct {
	Extra map[string]any `yaml:"extra"`
}

type TTSConfig struct {
	Speaker     string      `yaml:"speaker"`
	AudioConfig AudioConfig `yaml:"audio_config"`
}

type AudioConfig struct {
	Channel    int    `yaml:"channel"`
	Format     string `yaml:"format"`
	SampleRate int    `yaml:"sample_rate"`
}

type DialogConfig struct {
	BotName       string          `yaml:"bot_name"`
	SystemRole    string          `yaml:"system_role"`
	SpeakingStyle string          `yaml:"speaking_style"`
	Location      *LocationConfig `yaml:"location"`
	Extra         map[string]any  `yaml:"extra"`
}

type LocationConfig struct {
	City string `yaml:"city"`
}

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()
	return parse(f)
}

func parse(r io.Reader) (*Config, error) {
	var cfg Config
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func MustLoad(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		panic(err)
	}
	return cfg
}

func (c *Config) Validate() error {
	if c.Server.Port == 0 {
		return fmt.Errorf("server.port is required")
	}
	if c.Server.Host == "" {
		return fmt.Errorf("server.host is required")
	}
	if err := c.API.Validate(); err != nil {
		return err
	}
	if err := c.Session.Validate(); err != nil {
		return err
	}
	return nil
}

func (api APIConfig) Validate() error {
	switch {
	case api.URL == "":
		return fmt.Errorf("api.url is required")
	case api.AppID == "":
		return fmt.Errorf("api.app_id is required")
	case api.AppKey == "":
		return fmt.Errorf("api.app_key is required")
	case api.ResourceID == "":
		return fmt.Errorf("api.resource_id is required")
	case api.AccessKey == "":
		return fmt.Errorf("api.access_key is required")
	}
	return nil
}

func (s *SessionConfig) Validate() error {
	if s.TTS.Speaker == "" {
		return fmt.Errorf("session.tts.speaker is required")
	}
	if s.TTS.AudioConfig.SampleRate == 0 {
		return fmt.Errorf("session.tts.audio_config.sample_rate is required")
	}
	if s.TTS.AudioConfig.Channel == 0 {
		return fmt.Errorf("session.tts.audio_config.channel is required")
	}
	if s.TTS.AudioConfig.Format == "" {
		s.TTS.AudioConfig.Format = "pcm"
	}
	if s.Dialog.BotName == "" {
		return fmt.Errorf("session.dialog.bot_name is required")
	}
	if s.Dialog.SystemRole == "" {
		return fmt.Errorf("session.dialog.system_role is required")
	}
	return nil
}

func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

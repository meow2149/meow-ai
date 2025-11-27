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
	Extra ASRExtraConfig `yaml:"extra"`
}

type ASRExtraConfig struct {
	EndSmoothWindowMS int  `yaml:"end_smooth_window_ms"`
	EnableCustomVAD   bool `yaml:"enable_custom_vad"`
	EnableASRTwoPass  bool `yaml:"enable_asr_twopass"`
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
	DialogID          string          `yaml:"dialog_id"`
	BotName           string          `yaml:"bot_name"`
	SystemRole        string          `yaml:"system_role"`
	SpeakingStyle     string          `yaml:"speaking_style"`
	CharacterManifest string          `yaml:"character_manifest"`
	Location          *LocationConfig `yaml:"location"`
	Extra             DialogExtra     `yaml:"extra"`
}

type DialogExtra struct {
	StrictAudit              bool   `yaml:"strict_audit"`
	AuditResponse            string `yaml:"audit_response"`
	EnableVolcWebsearch      bool   `yaml:"enable_volc_websearch"`
	VolcWebsearchType        string `yaml:"volc_websearch_type"`
	VolcWebsearchAPIKey      string `yaml:"volc_websearch_api_key"`
	VolcWebsearchResultCount int    `yaml:"volc_websearch_result_count"`
	VolcWebsearchNoResultMsg string `yaml:"volc_websearch_no_result_message"`
	InputMod                 string `yaml:"input_mod"`
	Model                    string `yaml:"model"`
	RecvTimeout              int    `yaml:"recv_timeout"`
}

type LocationConfig struct {
	Longitude  float64 `yaml:"longitude"`
	Latitude   float64 `yaml:"latitude"`
	City       string  `yaml:"city"`
	Country    string  `yaml:"country"`
	Province   string  `yaml:"province"`
	District   string  `yaml:"district"`
	Town       string  `yaml:"town"`
	CountryISO string  `yaml:"country_code"`
	Address    string  `yaml:"address"`
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
	if len([]rune(s.Dialog.BotName)) > 20 {
		return fmt.Errorf("session.dialog.bot_name cannot exceed 20 characters")
	}
	if err := s.ASR.Extra.validate(); err != nil {
		return err
	}
	if err := s.Dialog.validate(); err != nil {
		return err
	}
	return nil
}

func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

func (e *ASRExtraConfig) validate() error {
	if e.EndSmoothWindowMS == 0 {
		e.EndSmoothWindowMS = 1500
	}
	if e.EndSmoothWindowMS < 500 || e.EndSmoothWindowMS > 50000 {
		return fmt.Errorf("session.asr.extra.end_smooth_window_ms must be between 500 and 50000")
	}
	return nil
}

func (d *DialogConfig) validate() error {
	if d.Location != nil {
		d.Location.setDefaults()
	}
	if d.Extra.VolcWebsearchType == "" {
		d.Extra.VolcWebsearchType = "web_summary"
	}
	if d.Extra.VolcWebsearchResultCount == 0 {
		d.Extra.VolcWebsearchResultCount = 10
	}
	if d.Extra.VolcWebsearchResultCount > 10 {
		return fmt.Errorf("session.dialog.extra.volc_websearch_result_count cannot exceed 10")
	}
	if d.Extra.Model == "" {
		d.Extra.Model = "O"
	}
	if d.Extra.RecvTimeout == 0 {
		d.Extra.RecvTimeout = 10
	}
	if d.Extra.RecvTimeout < 10 || d.Extra.RecvTimeout > 120 {
		return fmt.Errorf("session.dialog.extra.recv_timeout must be between 10 and 120")
	}
	if d.Extra.InputMod == "" {
		d.Extra.InputMod = "audio"
	}
	return nil
}

func (l *LocationConfig) setDefaults() {
	if l.Country == "" {
		l.Country = "中国"
	}
	if l.CountryISO == "" {
		l.CountryISO = "CN"
	}
}

// Package config loads the strict TOML 1.1 config schema.
package config

import (
	"bytes"
	stderrors "errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

const SchemaVersion = "onemoreffmpeg.config/v1"

type Config struct {
	Schema   string         `toml:"schema"`
	Log      LogConfig      `toml:"log"`
	Network  NetworkConfig  `toml:"network"`
	Download DownloadConfig `toml:"download"`
}

type LogConfig struct {
	Level string `toml:"level"`
	JSON  bool   `toml:"json"`
	Color string `toml:"color"`
}

type NetworkConfig struct {
	Timeout     string `toml:"timeout"`
	MaxAttempts int    `toml:"max_attempts"`
	GitHubAPI   string `toml:"github_api"`
}

type DownloadConfig struct {
	MaxBytes     int64 `toml:"max_bytes"`
	MaxRedirects int   `toml:"max_redirects"`
	// Dir overrides the default downloads directory when non-empty.
	Dir string `toml:"dir"`
}

func Default() Config {
	return Config{
		Schema: SchemaVersion,
		Log: LogConfig{
			Level: "info",
			JSON:  false,
			Color: "auto",
		},
		Network: NetworkConfig{
			Timeout:     "30s",
			MaxAttempts: 3,
			GitHubAPI:   "https://api.github.com",
		},
		Download: DownloadConfig{
			MaxBytes:     512 << 20,
			MaxRedirects: 5,
			Dir:          "",
		},
	}
}

func Load(path string) (Config, error) {
	if path == "" {
		return Default(), nil
	}
	f, err := os.Open(path)
	if err != nil {
		return Config{}, apperrors.New(apperrors.E7007, "config path invalid or unreadable").WithCause(err.Error())
	}
	defer f.Close()
	if fi, statErr := f.Stat(); statErr == nil && fi.IsDir() {
		return Config{}, apperrors.New(apperrors.E7007, "config path is a directory")
	}
	return Parse(f)
}

// LoadOptional loads path when it exists and returns defaults otherwise.
// A missing default config file is not an error; an unreadable path is.
func LoadOptional(path string) (Config, error) {
	if strings.TrimSpace(path) == "" {
		return Default(), nil
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return Config{}, apperrors.New(apperrors.E7007, "config path invalid or unreadable").WithCause(err.Error())
	}
	return Load(path)
}

func Parse(r io.Reader) (Config, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return Config{}, apperrors.New(apperrors.E7002, "config cannot be read").WithCause(err.Error())
	}

	var presence struct {
		Schema string `toml:"schema"`
	}
	if err := toml.Unmarshal(raw, &presence); err != nil {
		return Config{}, apperrors.New(apperrors.E7002, "TOML 1.1 syntax error").WithCause(err.Error())
	}
	if presence.Schema == "" {
		return Config{}, apperrors.New(apperrors.E7003, "config schema is required")
	}
	if presence.Schema != SchemaVersion {
		return Config{}, apperrors.New(apperrors.E7003, "config schema mismatch")
	}

	cfg := Default()
	dec := toml.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		// Syntax was already validated above, so the only remaining failures are
		// unknown keys or type/value mismatches; no message matching is needed.
		var strictErr *toml.StrictMissingError
		if stderrors.As(err, &strictErr) {
			return Config{}, apperrors.New(apperrors.E7004, "unknown key in config").WithCause(err.Error())
		}
		return Config{}, apperrors.New(apperrors.E7005, "config value type mismatch").WithCause(err.Error())
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	switch c.Log.Level {
	case "error", "warn", "info", "debug", "trace":
	default:
		return apperrors.New(apperrors.E7005, "invalid log level")
	}
	switch c.Log.Color {
	case "auto", "always", "never":
	default:
		return apperrors.New(apperrors.E7005, "invalid color mode")
	}
	if timeout, err := time.ParseDuration(c.Network.Timeout); err != nil || timeout <= 0 {
		return apperrors.New(apperrors.E7005, "invalid network timeout")
	}
	if c.Network.MaxAttempts < 1 || c.Network.MaxAttempts > 10 {
		return apperrors.New(apperrors.E7005, "network max_attempts must be 1..10")
	}
	if !isHTTPSURL(c.Network.GitHubAPI) {
		return apperrors.New(apperrors.E7005, "network github_api must be an https URL")
	}
	if c.Download.MaxBytes <= 0 {
		return apperrors.New(apperrors.E7005, "download max_bytes must be positive")
	}
	if c.Download.MaxRedirects < 0 || c.Download.MaxRedirects > 20 {
		return apperrors.New(apperrors.E7005, "download max_redirects must be 0..20")
	}
	if c.Download.Dir != "" && !filepath.IsAbs(c.Download.Dir) {
		return apperrors.New(apperrors.E7005, "download dir must be absolute")
	}
	return nil
}

func isHTTPSURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != ""
}

// Package config loads the strict TOML 1.1 config schema.
package config

import (
	"bytes"
	stderrors "errors"
	"io"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

const SchemaVersion = "onemoreffmpeg.config/v1"

type Config struct {
	Schema string    `toml:"schema"`
	Log    LogConfig `toml:"log"`
}

type LogConfig struct {
	Level string `toml:"level"`
	JSON  bool   `toml:"json"`
	Color string `toml:"color"`
}

func Default() Config {
	return Config{
		Schema: SchemaVersion,
		Log: LogConfig{
			Level: "info",
			JSON:  false,
			Color: "auto",
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

func Parse(r io.Reader) (Config, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return Config{}, apperrors.New(apperrors.E7002, "config cannot be read").WithCause(err.Error())
	}

	var presence struct {
		Schema string `toml:"schema"`
	}
	if err := toml.Unmarshal(raw, &presence); err != nil {
		return Config{}, classifyDecodeError(err)
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
		return Config{}, classifyDecodeError(err)
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func classifyDecodeError(err error) error {
	var strictErr *toml.StrictMissingError
	if stderrors.As(err, &strictErr) {
		return apperrors.New(apperrors.E7004, "unknown key in config").WithCause(err.Error())
	}
	msg := err.Error()
	if strings.Contains(msg, "cannot decode") || strings.Contains(msg, "incompatible") {
		return apperrors.New(apperrors.E7005, "config value type mismatch").WithCause(msg)
	}
	return apperrors.New(apperrors.E7002, "TOML 1.1 syntax error").WithCause(msg)
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
	return nil
}

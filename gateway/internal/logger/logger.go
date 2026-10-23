// Package logger holds the process-wide logging settings. The settings are
// configured once, at start-up; every package then asks for its own named
// logger, which shares those settings.
package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

// Format of the emitted records.
type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)

// Settings are shared by every logger in the process.
type Settings struct {
	// Level: debug | info | warn | error.
	Level string
	// Format: json | text.
	Format Format
	// Service is attached to every record as the "service" attribute.
	Service string
	// Output defaults to os.Stdout.
	Output io.Writer
}

// DefaultSettings are used when Init was not called.
func DefaultSettings(service string) Settings {
	return Settings{Level: "info", Format: FormatJSON, Service: service, Output: os.Stdout}
}

var (
	once sync.Once
	root *slog.Logger
)

// Init installs the shared settings. Only the first call has an effect, so a
// stray call cannot reconfigure the process.
func Init(settings Settings) error {
	var err error
	once.Do(func() {
		var handler slog.Handler
		handler, err = newHandler(settings)
		if err != nil {
			return
		}
		root = slog.New(handler).With(slog.String("service", settings.Service))
		slog.SetDefault(root)
	})
	return err
}

// L returns the shared root logger.
func L() *slog.Logger {
	if root == nil {
		// Something logs before Init: fall back to the defaults rather than
		// losing the record.
		_ = Init(DefaultSettings("unknown"))
	}
	return root
}

// Named returns a logger for one module: own name, shared settings.
//
//	log := logger.Named("repository.library")
func Named(name string) *slog.Logger {
	return L().With(slog.String("logger", name))
}

func newHandler(settings Settings) (slog.Handler, error) {
	level, err := parseLevel(settings.Level)
	if err != nil {
		return nil, err
	}

	output := settings.Output
	if output == nil {
		output = os.Stdout
	}

	options := &slog.HandlerOptions{Level: level}
	switch settings.Format {
	case FormatText:
		return slog.NewTextHandler(output, options), nil
	case FormatJSON, "":
		return slog.NewJSONHandler(output, options), nil
	default:
		return nil, fmt.Errorf("unknown log format %q", settings.Format)
	}
}

func parseLevel(level string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unknown log level %q", level)
	}
}

// FromEnv builds the settings from the environment, mirroring the other
// services: LOG_LEVEL, LOG_FORMAT, SERVICE_NAME.
func FromEnv(defaultService string) Settings {
	settings := DefaultSettings(defaultService)
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		settings.Level = level
	}
	if format := os.Getenv("LOG_FORMAT"); format != "" {
		settings.Format = Format(format)
	}
	if service := os.Getenv("SERVICE_NAME"); service != "" {
		settings.Service = service
	}
	return settings
}

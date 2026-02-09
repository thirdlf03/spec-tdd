package logger

import (
	"io"
	"log/slog"
	"os"
)

// Logger wraps slog.Logger with application-specific methods
type Logger struct {
	*slog.Logger
}

// Config holds logger configuration
type Config struct {
	Level  slog.Level
	Format string // "json" or "text"
	Writer io.Writer
}

// DefaultConfig returns the default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:  slog.LevelInfo,
		Format: "text",
		Writer: os.Stderr,
	}
}

// New creates a new logger with the given configuration
func New(cfg *Config) *Logger {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: cfg.Level,
		// Add source location for debug level
		AddSource: cfg.Level == slog.LevelDebug,
	}

	switch cfg.Format {
	case "json":
		handler = slog.NewJSONHandler(cfg.Writer, opts)
	default:
		handler = slog.NewTextHandler(cfg.Writer, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// NewFromFlags creates a logger from common CLI flags
func NewFromFlags(debug bool, format string) *Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	return New(&Config{
		Level:  level,
		Format: format,
		Writer: os.Stderr,
	})
}

// WithComponent adds a component name to all log entries
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger: l.With(slog.String("component", component)),
	}
}

// WithFields adds structured fields to all log entries
func (l *Logger) WithFields(attrs ...slog.Attr) *Logger {
	args := make([]any, len(attrs))
	for i, a := range attrs {
		args[i] = a
	}
	return &Logger{
		Logger: l.With(args...),
	}
}

// Package logging provides structured logging for ripper.
//
//  1. LEVELS follow the severity/verbosity model:
//     - ERROR: Something failed. The operation cannot continue. Always shown.
//     - WARN:  Something unexpected happened, but we can recover. Always shown unless --quiet.
//     - INFO:  High-level operational progress ("downloading playlist X"). Default level.
//     - DEBUG: Troubleshooting info for developers/operators ("yt-dlp args: [...]").
//     - TRACE: Extremely verbose. Internal state dumps, raw I/O, loop iterations.
//
//  2. STRUCTURED FIELDS: Every log line carries machine-parseable key=value pairs.
//     Always include: component (which subsystem), and contextual IDs (playlist_id, video_id).
//
//  3. VERBOSITY is controlled by a single integer (-v N):
//     -v 0 = ERROR only  (same as --quiet)
//     -v 1 = +WARN
//     -v 2 = +INFO       (default, no flag needed)
//     -v 3 = +DEBUG
//     -v 4 = +TRACE
//
//  4. OUTPUT FORMAT:
//     - Human-readable text to stderr.
//     - JSON mode available via --log-format=json.
//
//  5. STDERR, not stdout: Logs go to stderr. stdout is reserved for program output/data.
//     This follows Unix convention.
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

// Custom log levels following slog's convention:
// slog defines Debug=-4, Info=0, Warn=4, Error=8.
// We add Trace below Debug.
const (
	LevelTrace slog.Level = -8 // Below Debug (-4)
	LevelDebug            = slog.LevelDebug
	LevelInfo             = slog.LevelInfo
	LevelWarn             = slog.LevelWarn
	LevelError            = slog.LevelError
)

// Format controls the output format of the logger.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// Config holds logger configuration.
type Config struct {
	// Verbosity maps to log levels: 0=ERROR, 1=WARN, 2=INFO, 3=DEBUG, 4=TRACE
	Verbosity int

	// Format controls output format: "text" (default) or "json"
	Format Format

	// Output destination (defaults to os.Stderr)
	Output io.Writer

	// Component is a top-level field added to every log line (e.g., "ripper", "pipeline")
	Component string

	// Version is the application version, added to every log line
	Version string
}

var (
	// globalConfig stores the current configuration
	globalConfig Config
	configMu     sync.RWMutex
)

// Setup initializes the global logger with the given configuration.
// Call this once during CLI initialization (cobra.OnInitialize).
func Setup(cfg Config) {
	configMu.Lock()
	defer configMu.Unlock()
	globalConfig = cfg

	if cfg.Output == nil {
		cfg.Output = os.Stderr
	}

	level := verbosityToLevel(cfg.Verbosity)

	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Replace the numeric level with human-readable names
			if a.Key == slog.LevelKey {
				lvl := a.Value.Any().(slog.Level)
				a.Value = slog.StringValue(LevelName(lvl))
			}
			// Use shorter timestamp format for text output
			if a.Key == slog.TimeKey && cfg.Format == FormatText {
				t := a.Value.Time()
				a.Value = slog.StringValue(t.Format(time.TimeOnly))
			}
			return a
		},
	}

	var handler slog.Handler
	switch cfg.Format {
	case FormatJSON:
		handler = slog.NewJSONHandler(cfg.Output, opts)
	default:
		handler = slog.NewTextHandler(cfg.Output, opts)
	}

	// Wrap with default attributes (component, version) — these appear on every line
	attrs := []slog.Attr{}
	if cfg.Component != "" {
		attrs = append(attrs, slog.String("component", cfg.Component))
	}
	if cfg.Version != "" {
		attrs = append(attrs, slog.String("version", cfg.Version))
	}

	if len(attrs) > 0 {
		handler = handler.WithAttrs(attrs)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// verbosityToLevel converts the -v N integer to a slog.Level.
//
// Mapping:
//
//	0 → ERROR  (--quiet / -v 0)
//	1 → WARN
//	2 → INFO   (default)
//	3 → DEBUG
//	4 → TRACE
func verbosityToLevel(v int) slog.Level {
	switch {
	case v <= 0:
		return LevelError
	case v == 1:
		return LevelWarn
	case v == 2:
		return LevelInfo
	case v == 3:
		return LevelDebug
	default:
		return LevelTrace
	}
}

// LevelName returns a human-readable label for a log level.
func LevelName(l slog.Level) string {
	switch {
	case l <= LevelTrace:
		return "TRACE"
	case l <= LevelDebug:
		return "DEBUG"
	case l <= LevelInfo:
		return "INFO"
	case l <= LevelWarn:
		return "WARN"
	default:
		return "ERROR"
	}
}

// Component returns a child logger with the given component field.
// Use this to create per-package loggers:
//
//	var log = logging.Component("pipeline")
//	log.Info("processing playlist", "id", p.ID)
func Component(name string) *slog.Logger {
	return slog.Default().With("component", name)
}

// Trace logs at TRACE level (below DEBUG). Use for raw I/O, iteration dumps,
// internal state that's only useful when chasing the hardest bugs.
func Trace(msg string, args ...any) {
	slog.Log(context.Background(), LevelTrace, msg, args...)
}

// TraceContext logs at TRACE level with a context.
func TraceContext(ctx context.Context, msg string, args ...any) {
	slog.Log(ctx, LevelTrace, msg, args...)
}

// GetVerbosity returns the current verbosity level.
func GetVerbosity() int {
	configMu.RLock()
	defer configMu.RUnlock()
	return globalConfig.Verbosity
}

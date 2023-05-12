package logger

import (
	"io"
	std_log "log"
	"os"
	"strings"
	"time"

	"github.com/kacheio/kache/pkg/config"
	"github.com/natefinch/lumberjack"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	// Supress logs before setup.
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
}

// InitLogger initializes the logger.
func InitLogger(cfg *config.Log) {

	// configure log format
	format := initFormat(cfg)

	// configure log level
	level := initLevel(cfg)

	// create logger
	ctx := zerolog.New(format).With().Timestamp()
	if level <= zerolog.DebugLevel {
		// add caller info for Debug and Trace
		ctx = ctx.Caller()
	}

	log.Logger = ctx.Logger().Level(level)
	zerolog.DefaultContextLogger = &log.Logger
	zerolog.SetGlobalLevel(level)

	// configure standard log
	std_log.SetFlags(std_log.Lshortfile | std_log.LstdFlags)
}

// initFormat initializes the log format from
// config, returns a writer.
func initFormat(cfg *config.Log) io.Writer {
	var w io.Writer = os.Stderr

	if cfg != nil && cfg.FilePath != "" {
		// write logs to rolling files
		_, _ = os.Create(cfg.FilePath)
		w = &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   true,
		}
	}

	if cfg == nil || cfg.Format != "json" {
		// write logs to console
		w = zerolog.ConsoleWriter{
			Out:        w,
			TimeFormat: time.RFC3339,
			NoColor:    true,
		}
	}

	return w
}

// initLevel initializes the log level from config.
func initLevel(cfg *config.Log) zerolog.Level {
	level := "info"

	if cfg != nil && cfg.Level != "" {
		level = strings.ToLower(cfg.Level)
	}

	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		log.Error().Err(err).
			Str("logLevel", level).
			Msg("Unspecified or invalid log level, setting level to default (ERROR)...")

		logLevel = zerolog.ErrorLevel
	}

	return logLevel
}

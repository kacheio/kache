package logger

import (
	"io"
	std_log "log"
	"os"
	"strings"
	"time"

	"github.com/natefinch/lumberjack"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Config holds the configuration for the logger.
type Config struct {
	Level  string `yaml:"level,omitempty"`
	Format string `yaml:"format,omitempty"`

	FilePath   string `yaml:"filePath,omitempty"`
	MaxSize    int    `yaml:"maxSize,omitempty"`
	MaxAge     int    `yaml:"maxAge,omitempty"`
	MaxBackups int    `yaml:"maxBackups,omitempty"`
	Compress   bool   `yaml:"compress,omitempty"`
}

func init() {
	// Supress logs before setup.
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
}

// InitLogger initializes the logger.
func InitLogger(config *Config) {

	// configure log format
	format := initFormat(config)

	// configure log level
	level := initLevel(config)

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
func initFormat(config *Config) io.Writer {
	var w io.Writer = os.Stderr

	if config != nil && config.FilePath != "" {
		// write logs to rolling files
		_, _ = os.Create(config.FilePath)
		w = &lumberjack.Logger{
			Filename:   config.FilePath,
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   true,
		}
	}

	if config == nil || config.Format != "json" {
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
func initLevel(config *Config) zerolog.Level {
	level := "info"

	if config != nil && config.Level != "" {
		level = strings.ToLower(config.Level)
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

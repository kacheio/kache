// MIT License
//
// Copyright (c) 2023 kache.io
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

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
			NoColor:    (cfg != nil && (!cfg.Color || len(cfg.FilePath) > 0)),
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

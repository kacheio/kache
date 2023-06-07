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
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/kacheio/kache/pkg/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestLoggerInit(t *testing.T) {
	out := &bytes.Buffer{}

	w := zerolog.ConsoleWriter{Out: out, TimeFormat: time.RFC3339, NoColor: true}
	log := zerolog.New(w).With().Logger()

	log.Info().Msg("supressed")

	InitLogger(nil)

	log.Info().Msg("test")

	if got, want := out.String(), "<nil> INF test\n"; got != want {
		t.Errorf("invalid log output:\ngot:  %v\nwant: %v", got, want)
	}
}

func ExampleInitLogger() {
	_stderr := os.Stderr
	os.Stderr = os.Stdout

	location, _ := time.LoadLocation("UTC")
	time.Local = location

	zerolog.TimestampFunc = func() time.Time {
		return time.Unix(0, 0).UTC()
	}

	// Test log format

	// no config should log console
	InitLogger(nil)
	log.Info().Msg("test nil")

	// empty format should log console
	InitLogger(&config.Log{Format: ""})
	log.Info().Msg("test empty")

	// common format should log console
	InitLogger(&config.Log{Format: "common"})
	log.Info().Msg("test common")

	// json format should log json
	InitLogger(&config.Log{Format: "json"})
	log.Info().Msg("test json")

	// Test log level

	// info level should not log debug
	InitLogger(&config.Log{Level: "info"})
	log.Info().Msg("test level info")
	log.Debug().Msg("test level info -- ignored")

	// debug level should add caller
	InitLogger(&config.Log{Level: "debug"})
	log.Info().Msg("test level debug")

	// Output:
	// 1970-01-01T00:00:00Z INF test nil
	// 1970-01-01T00:00:00Z INF test empty
	// 1970-01-01T00:00:00Z INF test common
	// {"level":"info","time":"1970-01-01T00:00:00Z","message":"test json"}
	// 1970-01-01T00:00:00Z INF test level info
	// 1970-01-01T00:00:00Z INF logger_test.go:91 > test level debug

	os.Stderr = _stderr
}

package logger

import (
	"bytes"
	"os"
	"testing"
	"time"

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
	InitLogger(&Config{Format: ""})
	log.Info().Msg("test empty")

	// common format should log console
	InitLogger(&Config{Format: "common"})
	log.Info().Msg("test common")

	// json format should log json
	InitLogger(&Config{Format: "json"})
	log.Info().Msg("test json")

	// Test log level

	// info level should not log debug
	InitLogger(&Config{Level: "info"})
	log.Info().Msg("test level info")
	log.Debug().Msg("test level info -- ignored")

	// debug level should add caller
	InitLogger(&Config{Level: "debug"})
	log.Info().Msg("test level debug")

	// Output:
	// 1970-01-01T00:00:00Z INF test nil
	// 1970-01-01T00:00:00Z INF test empty
	// 1970-01-01T00:00:00Z INF test common
	// {"level":"info","time":"1970-01-01T00:00:00Z","message":"test json"}
	// 1970-01-01T00:00:00Z INF test level info
	// 1970-01-01T00:00:00Z INF logger_test.go:68 > test level debug

	os.Stderr = _stderr
}

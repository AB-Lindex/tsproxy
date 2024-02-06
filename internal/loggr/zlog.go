package loggr

import (
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Logger struct {
	enabled bool
	zlogger zerolog.Logger
}

func init() {
	if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05.999",
		})
	} else {
		zerolog.TimeFieldFormat = time.RFC3339Nano
		zerolog.TimestampFieldName = "ts"
	}
}

func New() logr.LogSink {
	return Logger{
		enabled: true,
		zlogger: log.Logger,
	}
}

func (l Logger) Enabled(_ int) bool {
	return l.enabled
}

func (l Logger) Init(_ logr.RuntimeInfo) {
}

func (l Logger) print(e *zerolog.Event, msg string, keysAndValues ...interface{}) {

	for i := 0; i < len(keysAndValues); i += 2 {
		e.Any(keysAndValues[i].(string), keysAndValues[i+1])
	}
	e.Msg(msg)
}

func (l Logger) Info(_ int, msg string, keysAndValues ...interface{}) {
	if !l.enabled {
		return
	}
	e := log.Info()
	l.print(e, msg, keysAndValues...)
}

func (l Logger) Error(err error, msg string, keysAndValues ...interface{}) {
	if !l.enabled {
		return
	}
	e := log.Error().Err(err)
	l.print(e, msg, keysAndValues...)
}

func (l Logger) V(level int) logr.LogSink {
	return l
}

func (l Logger) WithValues(keysAndValues ...interface{}) logr.LogSink {
	var l2 = l.zlogger.With().Logger()

	l2.UpdateContext(func(c zerolog.Context) zerolog.Context {
		for i := 0; i < len(keysAndValues); i += 2 {
			c = c.Any(keysAndValues[i].(string), keysAndValues[i+1])
		}
		return c
	})

	return Logger{
		enabled: l.enabled,
		zlogger: l2,
	}
}

func (l Logger) WithName(name string) logr.LogSink {
	var l2 = l.zlogger.With().Logger()

	l2.UpdateContext(func(c zerolog.Context) zerolog.Context {
		c = c.Str("name", name)
		return c
	})

	return Logger{
		enabled: l.enabled,
		zlogger: l2,
	}
}

func (l Logger) WithCallDepth(depth int) logr.LogSink {
	return l
}

func (l Logger) WithCallStackHelper() (func(), Logger) {
	return func() {}, l
}

func (l Logger) IsZero() bool {
	return false
}

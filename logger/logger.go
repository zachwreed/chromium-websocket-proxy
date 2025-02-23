package logger

import (
	"chromium-websocket-proxy/config"
	"context"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"sync"
)

var once sync.Once

var l zerolog.Logger

const (
	BrowserIdTrackingKey      = "browserId"
	BrowserProfileTrackingKey = "browserProfile"
	SessionIdTrackingKey      = "sessionId"
)

type TracingHook struct{}

// Run adds specific context keys to the log event if they exist in ctx: sessionId, browserId, and browserProfile.
func (h TracingHook) Run(e *zerolog.Event, _ zerolog.Level, _ string) {
	ctx := e.GetCtx()

	h.addKeyToEventIfExists(e, ctx, SessionIdTrackingKey)
	h.addKeyToEventIfExists(e, ctx, BrowserIdTrackingKey)
	h.addKeyToEventIfExists(e, ctx, BrowserProfileTrackingKey)
}

func (h TracingHook) addKeyToEventIfExists(e *zerolog.Event, eCtx context.Context, ctxKey string) {
	val := eCtx.Value(ctxKey)
	if val != nil {
		switch val.(type) {
		case string:
			e.Str(ctxKey, val.(string))

		case uuid.UUID:
			if val != uuid.Nil {
				e.Str(ctxKey, val.(uuid.UUID).String())
			}
		}
	}
}

func Get() zerolog.Logger {
	once.Do(func() {
		zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

		conf := config.Get().GetLoggerConfig()

		var output io.Writer = zerolog.ConsoleWriter{
			Out: os.Stdout,
		}

		if len(conf.LogFilePath) > 0 {
			fileLogger := &lumberjack.Logger{
				Filename:   conf.LogFilePath,
				MaxBackups: 10,
				MaxAge:     14,
				Compress:   true,
			}
			output = zerolog.MultiLevelWriter(os.Stderr, fileLogger)
		}

		l = zerolog.New(output).
			Level(conf.LogLevel).
			With().
			Timestamp().
			Logger()

		l = l.Hook(TracingHook{})
		zerolog.DefaultContextLogger = &l
	})
	return l
}

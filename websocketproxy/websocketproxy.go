package websocketproxy

import (
	"chromium-websocket-proxy/logger"
	"context"
	"github.com/rs/zerolog"
	"golang.org/x/time/rate"
	"io"
	"nhooyr.io/websocket"
	"time"
)

type Types string

const (
	Chrome Types = "chrome"
	Client Types = "client"
)

// IWebsocketProxyConnection - only expose needed connection functions
type IWebsocketProxyConnection interface {
	Writer(context.Context, websocket.MessageType) (io.WriteCloser, error)
	Reader(context.Context) (websocket.MessageType, io.Reader, error)
}

type WebsocketProxy struct {
	rConn             IWebsocketProxyConnection
	rContext          context.Context
	rType             Types
	rLimiter          *rate.Limiter
	wConn             IWebsocketProxyConnection
	wContext          context.Context
	waitTimeoutInSecs int
}

func NewWebsocketProxy(
	rConn IWebsocketProxyConnection,
	rContext context.Context,
	rType Types,
	rlimiter *rate.Limiter,
	waitTimeoutInSecs int,
) *WebsocketProxy {
	wp := &WebsocketProxy{
		rConn:             rConn,
		rContext:          rContext,
		rType:             rType,
		rLimiter:          rlimiter,
		waitTimeoutInSecs: waitTimeoutInSecs,
	}
	return wp
}

func (wp *WebsocketProxy) SetWriteConnection(
	wConn IWebsocketProxyConnection,
	wContext context.Context,
) {
	wp.wConn = wConn
	wp.wContext = wContext
}

func getTruncatedString(b *[]byte) string {
	s := string(*b)
	if len(s) > 200 {
		s = s[:200] + "..."
	}
	return s
}

func (wp *WebsocketProxy) read() (websocket.MessageType, []byte, error) {
	msgT, reader, err := wp.rConn.Reader(wp.rContext)
	if err != nil {
		return -1, nil, err
	}

	log := logger.Get()

	msg, err := io.ReadAll(reader)
	if err != nil {
		log.Error().
			Ctx(wp.rContext).
			Err(err).
			Str("messageBody", getTruncatedString(&msg)).
			Str("from", string(wp.rType)).
			Msg("error reading message")
		return msgT, msg, err
	}

	// don't bother with truncation unless debug level is set
	if log.GetLevel() == zerolog.DebugLevel {
		go func() {
			log.Debug().
				Ctx(wp.rContext).
				Str("messageBody", getTruncatedString(&msg)).
				Str("from", string(wp.rType)).
				Msg("received message")
		}()
	}
	return msgT, msg, nil
}

func (wp *WebsocketProxy) write(msgT websocket.MessageType, msg *[]byte) (err error) {
	writer, err := wp.wConn.Writer(wp.wContext, msgT)
	if err != nil {
		return err
	}
	_, err = writer.Write(*msg)
	if err != nil {
		return err
	}
	return writer.Close()
}

func (wp *WebsocketProxy) Proxy() error {
	rCtxWithTimeout, cancel := context.WithTimeout(
		wp.rContext,
		time.Second*time.Duration(wp.waitTimeoutInSecs),
	)
	defer cancel()

	// wait
	err := wp.rLimiter.Wait(rCtxWithTimeout)
	if err != nil {
		return err
	}

	msgT, msg, err := wp.read()
	if err != nil {
		return err
	}
	return wp.write(msgT, &msg)
}

package websocketproxy

import (
	"chromium-websocket-proxy/test/mocks/writeclosermock"
	"chromium-websocket-proxy/test/mocks/wsconnmock"
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
	"io"
	"nhooyr.io/websocket"
	"strings"
	"testing"
	"time"
)

func TestProxyReadAndWrite(t *testing.T) {
	expectedBody := `{ "message": "ok" }`
	var rWriteBytes []byte

	mockRWriteCloser := writeclosermock.NewMock(
		func(bytes []byte) (n int, err error) {
			return -1, errors.New("r WriteCloser should not be written to")
		},
		func() error {
			return errors.New("r WriteCloser should not be closed")
		},
	)

	mockRConn := wsconnmock.NewMock(
		func(ctx context.Context) (websocket.MessageType, io.Reader, error) {
			return websocket.MessageText, strings.NewReader(expectedBody), nil
		},
		func(ctx context.Context, messageType websocket.MessageType) (io.WriteCloser, error) {
			return mockRWriteCloser, errors.New("r conn should not be written to")
		},
	)

	mockWWriteCloser := writeclosermock.NewMock(
		func(bytes []byte) (n int, err error) {
			rWriteBytes = bytes
			return 0, nil
		},
		func() error {
			return nil
		},
	)

	mockWConn := wsconnmock.NewMock(
		func(ctx context.Context) (websocket.MessageType, io.Reader, error) {
			return websocket.MessageText, strings.NewReader(""), errors.New("w conn should not be read from")
		},
		func(ctx context.Context, messageType websocket.MessageType) (io.WriteCloser, error) {
			return mockWWriteCloser, nil
		},
	)

	limiter := rate.NewLimiter(rate.Every(time.Millisecond*10), 10)

	// init websocket proxy
	wp := NewWebsocketProxy(
		mockRConn,
		context.Background(),
		Client,
		limiter,
		10,
	)
	wp.SetWriteConnection(mockWConn, context.Background())

	err := wp.Proxy()

	assert.Nil(t, err)
	assert.Equal(t, expectedBody, string(rWriteBytes))
}

func TestProxyReadFailureDoesNotWrite(t *testing.T) {
	expectedBody := `{ "message": "ok" }`
	expectedErr := errors.New("reader error")
	var rWriteBytes []byte

	mockRWriteCloser := writeclosermock.NewMock(
		func(bytes []byte) (n int, err error) {
			return -1, errors.New("r WriteCloser should not be written to")
		},
		func() error {
			return errors.New("r WriteCloser should not be closed")
		},
	)

	mockRConn := wsconnmock.NewMock(
		func(ctx context.Context) (websocket.MessageType, io.Reader, error) {
			return websocket.MessageText, strings.NewReader(expectedBody), expectedErr
		},
		func(ctx context.Context, messageType websocket.MessageType) (io.WriteCloser, error) {
			return mockRWriteCloser, errors.New("r conn should not be written to")
		},
	)

	mockWWriteCloser := writeclosermock.NewMock(
		func(bytes []byte) (n int, err error) {
			rWriteBytes = bytes
			return 0, nil
		},
		func() error {
			return nil
		},
	)

	mockWConn := wsconnmock.NewMock(
		func(ctx context.Context) (websocket.MessageType, io.Reader, error) {
			return websocket.MessageText, strings.NewReader(""), errors.New("w conn should not be read from")
		},
		func(ctx context.Context, messageType websocket.MessageType) (io.WriteCloser, error) {
			return mockWWriteCloser, nil
		},
	)

	limiter := rate.NewLimiter(rate.Every(time.Millisecond*10), 10)

	// init websocket proxy
	wp := NewWebsocketProxy(
		mockRConn,
		context.Background(),
		Client,
		limiter,
		10,
	)
	wp.SetWriteConnection(mockWConn, context.Background())

	err := wp.Proxy()

	assert.Equal(t, expectedErr, err)
	assert.Empty(t, rWriteBytes)
}

func TestProxyWriteFailureReturnsError(t *testing.T) {
	expectedBody := `{ "message": "ok" }`
	expectedErr := errors.New("write error")
	var rWriteBytes []byte

	mockRWriteCloser := writeclosermock.NewMock(
		func(bytes []byte) (n int, err error) {
			return -1, errors.New("r WriteCloser should not be written to")
		},
		func() error {
			return errors.New("r WriteCloser should not be closed")
		},
	)

	mockRConn := wsconnmock.NewMock(
		func(ctx context.Context) (websocket.MessageType, io.Reader, error) {
			return websocket.MessageText, strings.NewReader(expectedBody), nil
		},
		func(ctx context.Context, messageType websocket.MessageType) (io.WriteCloser, error) {
			return mockRWriteCloser, errors.New("r conn should not be written to")
		},
	)

	mockWWriteCloser := writeclosermock.NewMock(
		func(bytes []byte) (n int, err error) {
			rWriteBytes = bytes
			return 0, expectedErr
		},
		func() error {
			return nil
		},
	)

	mockWConn := wsconnmock.NewMock(
		func(ctx context.Context) (websocket.MessageType, io.Reader, error) {
			return websocket.MessageText, strings.NewReader(""), errors.New("w conn should not be read from")
		},
		func(ctx context.Context, messageType websocket.MessageType) (io.WriteCloser, error) {
			return mockWWriteCloser, nil
		},
	)

	limiter := rate.NewLimiter(rate.Every(time.Millisecond*10), 10)

	// init websocket proxy
	wp := NewWebsocketProxy(
		mockRConn,
		context.Background(),
		Client,
		limiter,
		10,
	)
	wp.SetWriteConnection(mockWConn, context.Background())

	err := wp.Proxy()

	assert.Equal(t, expectedErr, err)
	assert.Equal(t, expectedBody, string(rWriteBytes))
}

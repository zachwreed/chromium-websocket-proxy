package wsconnmock

import (
	"context"
	"io"
	"nhooyr.io/websocket"
)

type MockConn struct {
	// added mocked functions
	mockedReader func(context.Context) (websocket.MessageType, io.Reader, error)
	mockedWriter func(context.Context, websocket.MessageType) (io.WriteCloser, error)
}

func NewMock(
	reader func(context.Context) (websocket.MessageType, io.Reader, error),
	writer func(context.Context, websocket.MessageType) (io.WriteCloser, error),
) *MockConn {
	mc := MockConn{
		mockedReader: reader,
		mockedWriter: writer,
	}
	return &mc
}

func (mc *MockConn) Reader(c context.Context) (websocket.MessageType, io.Reader, error) {
	return mc.mockedReader(c)
}

func (mc *MockConn) Writer(c context.Context, mt websocket.MessageType) (io.WriteCloser, error) {
	return mc.mockedWriter(c, mt)
}

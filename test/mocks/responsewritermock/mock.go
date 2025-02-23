package responsewritermock

import "net/http"

type ResponseWriterStruct struct {
	header      func() http.Header
	write       func([]byte) (int, error)
	writeHeader func(int)
}

func NewMock() *ResponseWriterStruct {
	return &ResponseWriterStruct{}
}

func (rw *ResponseWriterStruct) Header() http.Header {
	return rw.header()
}

func (rw *ResponseWriterStruct) SetHeader(
	header func() http.Header,
) {
	rw.header = header
}

func (rw *ResponseWriterStruct) Write(b []byte) (int, error) {
	return rw.write(b)
}

func (rw *ResponseWriterStruct) SetWrite(
	write func([]byte) (int, error),
) {
	rw.write = write
}

func (rw *ResponseWriterStruct) WriteHeader(statusCode int) {
	rw.writeHeader(statusCode)
}

func (rw *ResponseWriterStruct) SetWriteHeader(
	writeHeader func(int),
) {
	rw.writeHeader = writeHeader
}

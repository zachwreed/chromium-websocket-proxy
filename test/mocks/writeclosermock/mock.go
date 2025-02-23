package writeclosermock

type MockWriteCloser struct {
	write func([]byte) (n int, err error)
	close func() error
}

func NewMock(
	write func([]byte) (n int, err error),
	close func() error,
) MockWriteCloser {
	return MockWriteCloser{
		write: write,
		close: close,
	}
}

func (m MockWriteCloser) Write(p []byte) (n int, err error) {
	return m.write(p)
}

func (m MockWriteCloser) Close() error {
	return m.close()
}

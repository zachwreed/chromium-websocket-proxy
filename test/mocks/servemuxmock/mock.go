package servemuxmock

import (
	"net/http"
)

type MockServeMux struct {
	serveHTTP  func(w http.ResponseWriter, r *http.Request)
	handleFunc func(pattern string, handler func(http.ResponseWriter, *http.Request))
	handler    func(r *http.Request) (h http.Handler, pattern string)
	handle     func(pattern string, handler http.Handler)
}

func NewMock() *MockServeMux {
	msm := MockServeMux{}
	return &msm
}

func (msm *MockServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	msm.serveHTTP(w, r)
}

func (msm *MockServeMux) SetServeHTTP(
	serveHTTP func(w http.ResponseWriter, r *http.Request),
) {
	msm.serveHTTP = serveHTTP
}

func (msm *MockServeMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	msm.handleFunc(pattern, handler)
}

func (msm *MockServeMux) SetHandleFunc(
	handleFunc func(pattern string, handler func(http.ResponseWriter, *http.Request)),
) {
	msm.handleFunc = handleFunc
}

func (msm *MockServeMux) Handler(r *http.Request) (h http.Handler, pattern string) {
	return msm.handler(r)
}

func (msm *MockServeMux) SetHandler(
	handler func(r *http.Request) (h http.Handler, pattern string),
) {
	msm.handler = handler
}

func (msm *MockServeMux) Handle(pattern string, handler http.Handler) {
	msm.handle(pattern, handler)
}

func (msm *MockServeMux) SetHandle(
	handle func(pattern string, handler http.Handler),
) {
	msm.handle = handle
}

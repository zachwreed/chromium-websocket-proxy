package chromemock

import (
	"chromium-websocket-proxy/config"
	"context"
	"github.com/google/uuid"
)

type MockChrome struct {
	ctx                 context.Context
	cancel              context.CancelFunc
	remoteDebuggingPort int
	debugUrl            string
	sessionId           uuid.UUID
	profile             string
	browserID           uuid.UUID
	port                int
	start               func() error
	isIdle              bool
	isNew               bool
	options             config.ChromeConfigOptions
	conf                config.ChromeConfig
}

func NewMock() *MockChrome {
	mc := MockChrome{}
	mc.start = func() error {
		return nil
	}
	return &mc
}

func (mc *MockChrome) IsIdle() bool {
	return mc.isIdle
}

func (mc *MockChrome) SetNotIdle() {
	mc.isIdle = false
}

func (mc *MockChrome) SetIdleOrStop() {
	mc.isIdle = true
}

func (mc *MockChrome) Ctx() context.Context {
	return mc.ctx
}

func (mc *MockChrome) DebugUrl() string {
	return mc.debugUrl
}

func (mc *MockChrome) SetDebugUrl(debugUrl string) {
	mc.debugUrl = debugUrl
}

func (mc *MockChrome) SessionId() uuid.UUID {
	return mc.sessionId
}

func (mc *MockChrome) Profile() string {
	return mc.profile
}

func (mc *MockChrome) SetSessionId(sessionId uuid.UUID) {
	mc.sessionId = sessionId
}

func (mc *MockChrome) BrowserID() uuid.UUID {
	return mc.browserID
}

func (mc *MockChrome) SetBrowserID(browserID uuid.UUID) {
	mc.browserID = browserID
}

func (mc *MockChrome) Port() int {
	return mc.port
}

func (mc *MockChrome) Stop() {}

func (mc *MockChrome) Start() error {
	return mc.start()
}
func (mc *MockChrome) SetStart(start func() error) {
	mc.start = start
}

func (mc *MockChrome) Options() config.ChromeConfigOptions {
	return mc.options
}

func (mc *MockChrome) Config() config.ChromeConfig {
	return mc.conf
}

func (mc *MockChrome) SetOptions(options config.ChromeConfigOptions) {
	mc.options = options
}

func (mc *MockChrome) StartTicker() {}

func (mc *MockChrome) PauseTicker() {}

func (mc *MockChrome) IsNew() bool { return mc.isNew }

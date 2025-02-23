package chromepoolmock

import (
	"chromium-websocket-proxy/chrome"
	"chromium-websocket-proxy/config"
	"github.com/chromedp/cdproto/target"
	"github.com/google/uuid"
)

type MockChromePool struct {
	createNewInstance     func(config.ChromeConfigOptions) error
	getAvailableChrome    func(uuid.UUID, config.ChromeConfigOptions) (chrome.IChrome, error)
	hasIdleChromeInstance bool
	isPoolAtCapacity      bool
}

func NewMock() *MockChromePool {
	mcp := MockChromePool{}
	return &mcp
}

func (mcp *MockChromePool) GetAvailableChrome(sessionId uuid.UUID, options config.ChromeConfigOptions) (*chrome.IChrome, error) {
	crm, err := mcp.getAvailableChrome(sessionId, options)
	return &crm, err
}

func (mcp *MockChromePool) SetGetAvailableChrome(
	getAvailableChrome func(uuid.UUID, config.ChromeConfigOptions) (chrome.IChrome, error),
) {
	mcp.getAvailableChrome = getAvailableChrome
}

func (mcp *MockChromePool) ShutDownPool() {}

func (mcp *MockChromePool) GetInstancePoolLen() int {
	return 1
}

func (mcp *MockChromePool) HasIdleChromeInstance() bool {
	return mcp.hasIdleChromeInstance
}

func (mcp *MockChromePool) IsPoolAtCapacity() bool {
	return mcp.isPoolAtCapacity
}

func (mcp *MockChromePool) GetChromeFirstPageTargetIDAtIndex(i int) (target.ID, error) {
	return "", nil
}

func (mcp *MockChromePool) CreateNewInstance(options config.ChromeConfigOptions) error {
	return mcp.createNewInstance(options)
}

package servemux

import (
	"chromium-websocket-proxy/config"
	"chromium-websocket-proxy/test/mocks/servemuxmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"net/http"
	"sync"
	"testing"
)

type ServeMuxTestSuite struct {
	suite.Suite
}

// run before each test
func (suite *ServeMuxTestSuite) SetupTest() {
	config.Once = sync.Once{}

	//chromeGet := func(_ uuid.UUID, _ string) (chrome.IChrome, error) {
	//	cm := chromemock.NewMock()
	//	return cm, nil
	//}
	//
	//chromePoolGet = func() chromepool.IChromePool {
	//	cp := chromepoolmock.NewMock()
	//	cp.SetGetAvailableChrome(chromeGet)
	//	return cp
	//}
}

func (suite *ServeMuxTestSuite) TestNewServer() {
	handleFuncInvoked := false

	smm := servemuxmock.NewMock()
	smm.SetHandleFunc(
		func(pattern string, handler func(http.ResponseWriter, *http.Request)) {
			handleFuncInvoked = true
		},
	)

	NewServeMux(smm)
	assert.True(suite.T(), handleFuncInvoked)
}

func TestLoggerSuite(t *testing.T) {
	suite.Run(t, new(ServeMuxTestSuite))
}

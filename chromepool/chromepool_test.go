package chromepool

import (
	"chromium-websocket-proxy/chrome"
	"chromium-websocket-proxy/config"
	"chromium-websocket-proxy/metrics"
	"chromium-websocket-proxy/test/mocks/chromemock"
	"chromium-websocket-proxy/test/testutils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"strconv"
	"sync"
	"testing"
	"time"
)

type ChromePoolTestSuite struct {
	suite.Suite
}

// run before each test
func (suite *ChromePoolTestSuite) SetupTest() {
	once = sync.Once{}
	config.Once = sync.Once{}
	freePort = func() (int, error) {
		return 1000, nil
	}
}

func (suite *ChromePoolTestSuite) TestNewChromePoolConfig() {
	debugPorts := []int{9000, 9001}
	testutils.SetDebugPortsForConfig(suite.T(), config.ChromeDebugPorts, debugPorts)
	suite.T().Setenv(config.MinBrowserInstances, strconv.FormatInt(1, 10))
	_ = metrics.Init()

	started := false

	start := func() error {
		started = true
		return nil
	}

	opt, _ := config.NewCreateOptions(&config.ChromeConfigOptionsPayload{
		Profile: "",
	})

	chromeCreator = func(payload chrome.CreateChromePayload) chrome.IChrome {
		cm := chromemock.NewMock()
		cm.SetStart(start)
		cm.SetOptions(opt)
		return cm
	}

	cp := Get()

	// validate first chrome instance exists in pool
	assert.Equal(suite.T(), cp.GetInstancePoolLen(), 1)
	assert.True(suite.T(), started)
	cp.ShutDownPool()
}

func (suite *ChromePoolTestSuite) TestChromeShutdownAndStart() {
	suite.T().Setenv(config.MaxBrowserInstances, strconv.FormatInt(1, 10))
	suite.T().Setenv(config.MinBrowserInstances, strconv.FormatInt(1, 10))
	_ = metrics.Init()

	type createPayload struct {
		debugUrl  string
		browserID uuid.UUID
	}

	firstChrome := createPayload{
		debugUrl:  "firstChromeDebugUrl",
		browserID: uuid.New(),
	}

	secondChrome := createPayload{
		debugUrl:  "secondChromeDebugUrl",
		browserID: uuid.New(),
	}

	shutdownChromeChannel := make(chan chrome.EventData, 1)
	startChromeChannel := make(chan createPayload, 1)
	stopChan := make(chan bool, 1)

	createChromeEventReceiver = func() chan chrome.EventData {
		return shutdownChromeChannel
	}

	createChromeEventReceiveStopper = func() chan bool {
		return stopChan
	}

	opt, _ := config.NewCreateOptions(&config.ChromeConfigOptionsPayload{
		Profile: "",
	})

	chromeCreator = func(payload chrome.CreateChromePayload) chrome.IChrome {
		for {
			select {
			case payload := <-startChromeChannel:
				cm := chromemock.NewMock()
				cm.SetDebugUrl(payload.debugUrl)
				cm.SetBrowserID(payload.browserID)
				cm.SetOptions(opt)
				cm.SetIdleOrStop()
				return cm
			case <-stopChan:
				break
			}
		}
	}

	startChromeChannel <- firstChrome
	cp := Get()

	// validate first chrome instance exists in pool
	assert.Equal(suite.T(), 1, cp.GetInstancePoolLen())
	crm, err := cp.GetAvailableChrome(uuid.New(), opt)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), firstChrome.debugUrl, (*crm).DebugUrl())

	shutdownChromeChannel <- chrome.EventData{
		BrowserID: firstChrome.browserID,
		EventType: chrome.ChromiumEventBrowserDestroyed,
	}
	startChromeChannel <- secondChrome

	// give CP enough time to shut down first and start second
	time.Sleep(time.Second * 2)

	assert.Equal(suite.T(), 1, cp.GetInstancePoolLen())
	crm, err = cp.GetAvailableChrome(uuid.New(), opt)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), secondChrome.debugUrl, (*crm).DebugUrl())

	stopChan <- true
	stopChan <- true
}

func (suite *ChromePoolTestSuite) TestHandleBrowserEvents() {
	debugPorts := []int{9000, 9001}
	testutils.SetDebugPortsForConfig(suite.T(), config.ChromeDebugPorts, debugPorts)
	suite.T().Setenv(config.MaxBrowserInstances, strconv.FormatInt(1, 10))
	suite.T().Setenv(config.MinBrowserInstances, strconv.FormatInt(0, 10))
	_ = metrics.Init()

	type createPayload struct {
		debugUrl  string
		browserID uuid.UUID
	}

	firstChrome := createPayload{
		debugUrl:  "firstChromeDebugUrl",
		browserID: uuid.New(),
	}

	secondChrome := createPayload{
		debugUrl:  "secondChromeDebugUrl",
		browserID: uuid.New(),
	}

	shutdownChromeChannel := make(chan chrome.EventData, 2)
	startChromeChannel := make(chan createPayload, 2)
	stopChan := make(chan bool, 1)

	createChromeEventReceiver = func() chan chrome.EventData {
		return shutdownChromeChannel
	}

	createChromeEventReceiveStopper = func() chan bool {
		return stopChan
	}

	chromeCreator = func(payload chrome.CreateChromePayload) chrome.IChrome {
		for {
			select {
			case payload := <-startChromeChannel:
				cm := chromemock.NewMock()
				cm.SetDebugUrl(payload.debugUrl)
				cm.SetBrowserID(payload.browserID)
				return cm
			case <-stopChan:
				break
			}
		}
	}

	startChromeChannel <- firstChrome
	cp := Get()

	assert.Equal(suite.T(), 0, cp.GetInstancePoolLen())
	crm1, err := cp.GetAvailableChrome(uuid.New(), config.ChromeConfigOptions{
		Profile: "",
	})
	assert.Nil(suite.T(), err)

	// validate first instance was created
	assert.Equal(suite.T(), 1, cp.GetInstancePoolLen())
	assert.Equal(suite.T(), firstChrome.debugUrl, (*crm1).DebugUrl())

	shutdownChromeChannel <- chrome.EventData{
		BrowserID: firstChrome.browserID,
		EventType: chrome.ChromiumEventBrowserDestroyed,
	}
	startChromeChannel <- secondChrome

	// give CP enough time to shut down first and start second
	time.Sleep(time.Second * 2)

	assert.Equal(suite.T(), 0, cp.GetInstancePoolLen())
	opt, _ := config.NewCreateOptions(&config.ChromeConfigOptionsPayload{
		Profile: "",
	})

	crm2, err := cp.GetAvailableChrome(uuid.New(), opt)
	assert.Nil(suite.T(), err)

	// validate second instance was created
	assert.Equal(suite.T(), 1, cp.GetInstancePoolLen())
	assert.Equal(suite.T(), secondChrome.debugUrl, (*crm2).DebugUrl())

	shutdownChromeChannel <- chrome.EventData{
		BrowserID: secondChrome.browserID,
		EventType: chrome.ChromiumEventBrowserDestroyed,
	}

	time.Sleep(time.Second * 2)
	assert.Equal(suite.T(), cp.GetInstancePoolLen(), 0)
	stopChan <- true
	stopChan <- true
}

func (suite *ChromePoolTestSuite) TestTagDebugUrl() {
	suite.T().Setenv(config.MinBrowserInstances, strconv.FormatInt(2, 10))
	_ = metrics.Init()

	type createPayload struct {
		DebugUrl  string
		SessionId uuid.UUID
	}

	crmPayload1 := createPayload{
		DebugUrl:  "crmPayload1",
		SessionId: uuid.New(),
	}

	crmPayload2 := createPayload{
		DebugUrl:  "crmPayload2",
		SessionId: uuid.New(),
	}

	startChromeChannel := make(chan createPayload, 2)
	stopChan := make(chan bool, 1)

	opt, _ := config.NewCreateOptions(&config.ChromeConfigOptionsPayload{
		Profile: "",
	})

	chromeCreator = func(payload chrome.CreateChromePayload) chrome.IChrome {
		for {
			select {
			case payload := <-startChromeChannel:
				cm := chromemock.NewMock()
				cm.SetDebugUrl(payload.DebugUrl)
				cm.SetIdleOrStop()
				cm.SetOptions(opt)
				return cm
			case <-stopChan:
				break
			}
		}
	}

	startChromeChannel <- crmPayload1
	startChromeChannel <- crmPayload2

	cp := Get()

	// validate instances are created
	assert.Equal(suite.T(), 2, cp.GetInstancePoolLen())

	crm1, err := cp.GetAvailableChrome(crmPayload1.SessionId, opt)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), crmPayload1.DebugUrl, (*crm1).DebugUrl())
	assert.Equal(suite.T(), crmPayload1.SessionId, (*crm1).SessionId())

	crm2, err := cp.GetAvailableChrome(crmPayload2.SessionId, opt)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), crmPayload2.DebugUrl, (*crm2).DebugUrl())
	assert.Equal(suite.T(), crmPayload2.SessionId, (*crm2).SessionId())

	stopChan <- true
	cp.ShutDownPool()
}

func (suite *ChromePoolTestSuite) TestThrowsErrorFromGetDebugUrlTagChromeLimit() {
	suite.T().Setenv(config.MinBrowserInstances, strconv.FormatInt(1, 10))
	suite.T().Setenv(config.MaxBrowserInstances, strconv.FormatInt(1, 10))
	_ = metrics.Init()

	chromeCreator = func(payload chrome.CreateChromePayload) chrome.IChrome {
		cm := chromemock.NewMock()
		cm.SetSessionId(payload.SessionId)
		return cm
	}

	cp := Get()

	// validate instances are created
	assert.Equal(suite.T(), 1, cp.GetInstancePoolLen())

	opt, _ := config.NewCreateOptions(&config.ChromeConfigOptionsPayload{
		Profile: "",
	})

	_, err := cp.GetAvailableChrome(uuid.New(), opt)
	assert.Error(suite.T(), err)
	cp.ShutDownPool()
}

func (suite *ChromePoolTestSuite) TestShutdownPool() {
	suite.T().Setenv(config.MinBrowserInstances, strconv.FormatInt(2, 10))
	suite.T().Setenv(config.MaxBrowserInstances, strconv.FormatInt(2, 10))
	_ = metrics.Init()

	chromeCreator = func(payload chrome.CreateChromePayload) chrome.IChrome {
		cm := chromemock.NewMock()
		return cm
	}

	cp := Get()

	// validate instances are created
	assert.Equal(suite.T(), 2, cp.GetInstancePoolLen())

	cp.ShutDownPool()

	assert.Equal(suite.T(), 0, cp.GetInstancePoolLen())
}

func TestLoggerSuite(t *testing.T) {
	suite.Run(t, new(ChromePoolTestSuite))
}

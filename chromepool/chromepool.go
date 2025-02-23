package chromepool

import (
	"chromium-websocket-proxy/chrome"
	"chromium-websocket-proxy/config"
	"chromium-websocket-proxy/logger"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"sync"
)

type IChromePool interface {
	GetAvailableChrome(sessionId uuid.UUID, options config.ChromeConfigOptions) (*chrome.IChrome, error)
	ShutDownPool()
	GetInstancePoolLen() int
	HasIdleChromeInstance() bool
	CreateNewInstance(options config.ChromeConfigOptions) error
	IsPoolAtCapacity() bool
}

type ChromePool struct {
	instancePool              []*chrome.IChrome
	instancePoolMutex         sync.RWMutex
	instancePoolHasIdleEl     bool
	availableDebuggingPorts   []int
	chromeEventReceiver       chan chrome.EventData
	chromeEventReceiveStopper chan bool
}

var once = sync.Once{}

var cp ChromePool

var createChromeEventReceiver = func() chan chrome.EventData {
	return make(chan chrome.EventData)
}

var createChromeEventReceiveStopper = func() chan bool {
	return make(chan bool)
}

func Get() IChromePool {
	once.Do(func() {
		conf := config.Get()
		log := logger.Get()

		cp = ChromePool{
			availableDebuggingPorts:   make([]int, len(conf.GetChromePoolConfig().DebugPorts)),
			chromeEventReceiver:       createChromeEventReceiver(),
			chromeEventReceiveStopper: createChromeEventReceiveStopper(),
		}
		copy(cp.availableDebuggingPorts[:], conf.GetChromePoolConfig().DebugPorts)

		for i := 0; i < conf.GetChromePoolConfig().MinBrowserInstances; i++ {
			err := cp.CreateNewInstance(conf.GetChromeConfig().DefaultOptions)
			if err != nil {
				log.Fatal().Err(err).Msg("unable to start chromium")
			}
		}
		go cp.chromiumEventReceiver()

		log.Info().Msg(fmt.Sprintf("initialized %d chromium browser(s)", cp.GetInstancePoolLen()))
	})
	return &cp
}

func (cp *ChromePool) CreateNewInstance(options config.ChromeConfigOptions) error {
	cp.instancePoolMutex.Lock()
	defer cp.instancePoolMutex.Unlock()
	_, err := cp.createChromeWLocked(uuid.Nil, options)
	return err
}

func (cp *ChromePool) chromiumEventReceiver() {
	for {
		select {
		case event := <-cp.chromeEventReceiver:
			switch event.EventType {
			case chrome.ChromiumEventBrowserDestroyed:
				cp.removeInstanceByBrowserId(event.BrowserID)
			case chrome.ChromiumEventTargetToDestroy:
				cp.removeInstanceByBrowserId(event.BrowserID)
			case chrome.ChromiumEventBrowserIdle:
				cp.checkInstanceByBrowserIdToRemove(event.BrowserID)
			}
		case <-cp.chromeEventReceiveStopper:
			return
		}
	}
}

func (cp *ChromePool) checkInstanceByBrowserIdToRemove(browserID uuid.UUID) {
	cp.instancePoolMutex.Lock()
	defer cp.instancePoolMutex.Unlock()
	crm, l := cp.getInstanceByBrowserIDLocked(browserID)
	if crm == nil {
		return
	}
	conf := config.Get()
	log := logger.Get()

	// this is an unused, default chrome instance when we are at min browser instances. Leave it be
	if (*crm).IsNew() &&
		l == config.Get().GetChromePoolConfig().MinBrowserInstances &&
		(*crm).Options().Hash == conf.GetChromeConfig().DefaultOptions.Hash {
		log.Debug().
			Str("browserId", (*crm).BrowserID().String()).
			Msg(fmt.Sprintf(
				"Unused browser at %s count has been idle for %s, pausing event handlers to reduce memory",
				config.MinBrowserInstances,
				config.ChromeBrowserAutoShutdownTimeoutInSecs,
			))
		(*crm).PauseTicker()
		return
	}
	log.Debug().
		Str("browserId", (*crm).BrowserID().String()).
		Msg(fmt.Sprintf(
			"Browser has been idle for %s, targetting for shutdown",
			config.ChromeBrowserAutoShutdownTimeoutInSecs,
		))
	cp.removeInstanceByBrowserIdWLocked(browserID)
}

func (cp *ChromePool) removeInstanceByBrowserId(browserID uuid.UUID) {
	cp.instancePoolMutex.Lock()
	defer cp.instancePoolMutex.Unlock()
	cp.removeInstanceByBrowserIdWLocked(browserID)
}

func (cp *ChromePool) IsPoolAtCapacity() bool {
	cp.instancePoolMutex.RLock()
	defer cp.instancePoolMutex.RUnlock()
	return cp.getInstancePoolLenLocked() >= config.Get().GetChromePoolConfig().MaxBrowserInstances
}

func (cp *ChromePool) HasIdleChromeInstance() bool {
	cp.instancePoolMutex.RLock()
	defer cp.instancePoolMutex.RUnlock()
	return cp.hasIdleChromeInstanceLocked()
}

func (cp *ChromePool) GetAvailableChrome(sessionId uuid.UUID, options config.ChromeConfigOptions) (*chrome.IChrome, error) {
	cp.instancePoolMutex.Lock()
	defer cp.instancePoolMutex.Unlock()

	ipLen := cp.getInstancePoolLenLocked()

	// create instance if none exist
	if ipLen == 0 {
		return cp.createChromeWLocked(sessionId, options)
	}

	// get existing idle browser with profile
	crm := cp.getIdleChromeLocked(options)
	if crm != nil {
		log := logger.Get()
		log.Info().
			Str("browserId", (*crm).BrowserID().String()).
			Str("sessionId", sessionId.String()).
			Msg("using idle chrome instance for session")
		(*crm).SetSessionId(sessionId)
		(*crm).SetNotIdle()
		(*crm).StartTicker()
		return crm, nil
	}

	// This is not default options. Attempt to create
	if options.Hash != config.Get().GetChromeConfig().DefaultOptions.Hash {
		return cp.createChromeWLocked(sessionId, options)
	}

	return nil, errors.New("no browser available for use")
}

func (cp *ChromePool) ShutDownPool() {
	cp.instancePoolMutex.Lock()
	defer cp.instancePoolMutex.Unlock()
	for {
		if cp.getInstancePoolLenLocked() == 0 {
			break
		}
		cp.removeInstanceAtIndexWLocked(0)
	}
	cp.chromeEventReceiveStopper <- true
	log := logger.Get()
	log.Info().Msg("gracefully shutdown chrome pool")
}

func (cp *ChromePool) GetInstancePoolLen() int {
	cp.instancePoolMutex.RLock()
	defer cp.instancePoolMutex.RUnlock()
	return cp.getInstancePoolLenLocked()
}

package chromepool

import (
	"chromium-websocket-proxy/chrome"
	"chromium-websocket-proxy/config"
	"chromium-websocket-proxy/logger"
	"chromium-websocket-proxy/metrics"
	"errors"
	"fmt"
	"github.com/google/uuid"
)

var chromeCreator = chrome.NewChrome

/**
 * write_locked.go contains all chromepool functions wrapped by a cp.instancePoolMutex.Lock().
 * The mutex should not be invoked in this file
 * All functions in this file should be named "...WLocked(...)"
 */

func (cp *ChromePool) createChromeWLocked(sessionId uuid.UUID, options config.ChromeConfigOptions) (*chrome.IChrome, error) {
	conf := config.Get()
	l := cp.getInstancePoolLenLocked()

	if l >= conf.GetChromePoolConfig().MaxBrowserInstances {
		return nil, errors.New(fmt.Sprintf("%s have been created already", config.MaxBrowserInstances))
	}

	port, err := cp.getAvailablePortLocked()
	if err != nil {
		return nil, err
	}

	crm := chromeCreator(chrome.CreateChromePayload{
		Port:          port,
		SessionId:     sessionId,
		EventReceiver: cp.chromeEventReceiver,
		Options:       options,
	})

	if err = crm.Start(); err != nil {
		return nil, err
	}
	cp.instancePool = append(cp.instancePool, &crm)
	metrics.Get().Remote.SetGauge(metrics.ChromeInstances, float32(len(cp.instancePool)))
	return &crm, nil
}

func (cp *ChromePool) removeInstanceAtIndexWLocked(i int) {
	log := logger.Get()
	log.Info().Str("browserId", (*cp.instancePool[i]).BrowserID().String()).Msg(fmt.Sprintf("destroying chrome browser instance at port %#v\n", (*cp.instancePool[i]).Port()))
	cp.availableDebuggingPorts = append(cp.availableDebuggingPorts, (*cp.instancePool[i]).Port())
	(*cp.instancePool[i]).Stop()
	cp.instancePool = append(cp.instancePool[:i], cp.instancePool[i+1:]...)
	metrics.Get().Remote.SetGauge(metrics.ChromeInstances, float32(len(cp.instancePool)))
}

func (cp *ChromePool) removeInstanceByBrowserIdWLocked(browserID uuid.UUID) {
	c := config.Get()
	minBrowserInstances := c.GetChromePoolConfig().MinBrowserInstances
	for i := 0; i < cp.getInstancePoolLenLocked(); i++ {
		crm := *cp.instancePool[i]
		if browserID == crm.BrowserID() {
			cp.removeInstanceAtIndexWLocked(i)

			if cp.getInstancePoolLenLocked() < minBrowserInstances {
				_, err := cp.createChromeWLocked(uuid.Nil, c.GetChromeConfig().DefaultOptions)
				if err != nil {
					log := logger.Get()
					log.Err(err).Msg("unable to start chrome browser")
				}
			}
			return
		}
	}
}

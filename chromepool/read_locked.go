package chromepool

import (
	"chromium-websocket-proxy/chrome"
	"chromium-websocket-proxy/config"
	"errors"
	"github.com/google/uuid"
	"github.com/phayes/freeport"
)

/**
 * read_locked.go contains all chromepool functions wrapped by a cp.instancePoolMutex.RLock().
 * The mutex should not be invoked in this file
 * All functions in this file should be named "...Locked(...)"
 */

var freePort = freeport.GetFreePort

func (cp *ChromePool) getIdleChromeLocked(options config.ChromeConfigOptions) *chrome.IChrome {
	for i := range cp.instancePool {
		if (*cp.instancePool[i]).IsIdle() && (*cp.instancePool[i]).Options().Hash == options.Hash {
			return cp.instancePool[i]
		}
	}
	return nil
}

func (cp *ChromePool) getInstanceByBrowserIDLocked(browserID uuid.UUID) (*chrome.IChrome, int) {
	l := cp.getInstancePoolLenLocked()
	for i := 0; i < l; i++ {
		if browserID == (*cp.instancePool[i]).BrowserID() {
			return cp.instancePool[i], l
		}
	}
	return nil, l
}

func (cp *ChromePool) hasIdleChromeInstanceLocked() bool {
	for i := range cp.instancePool {
		if (*cp.instancePool[i]).IsIdle() {
			return true
		}
	}
	return false
}

func (cp *ChromePool) hasChromeInstanceForHashLocked() bool {
	for i := range cp.instancePool {
		if (*cp.instancePool[i]).IsIdle() {
			return true
		}
	}
	return false
}

func (cp *ChromePool) getInstancePoolLenLocked() int {
	return len(cp.instancePool)
}

/* Wrapped by mutex lock already */
func (cp *ChromePool) getAvailablePortLocked() (int, error) {
	conf := config.Get()
	if !conf.GetChromePoolConfig().EnableAutoAssignDebugPort {
		if len(cp.availableDebuggingPorts) == 0 {
			return -1, errors.New("no available debug ports")
		}
		port, available := cp.availableDebuggingPorts[0], cp.availableDebuggingPorts[1:]
		cp.availableDebuggingPorts = available
		return port, nil
	}
	return freePort()
}

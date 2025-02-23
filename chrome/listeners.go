package chrome

import (
	"chromium-websocket-proxy/config"
	"chromium-websocket-proxy/logger"
	"fmt"
	"github.com/chromedp/cdproto/target"
	"time"
)

func (crm *Chrome) onBrowserEvent(v interface{}) {
	switch v := v.(type) {
	case *target.EventTargetDestroyed:
		go func() {
			if v.TargetID == crm.FirstPageTargetID() {
				crm.event.receiver <- EventData{
					BrowserID: crm.meta.browserID,
					EventType: ChromiumEventBrowserDestroyed,
				}
			}
		}()
	default:
		go func() {
			crm.event.lastEventTimestamp = time.Now()
		}()
	}
}

func (crm *Chrome) onTick() {
	for {
		select {
		case <-crm.event.ticker.C:
			log := logger.Get()
			now := time.Now()

			idleCheck := crm.event.lastEventTimestamp.Add(crm.conf.BrowserAutoSetIdleTimeoutInSecs)

			// browser is idle for BrowserAutoSetIdleTimeoutInSecs seconds
			if now.After(idleCheck) && !crm.IsIdle() {
				log.Debug().Ctx(crm.ea.ctx).Msg(fmt.Sprintf(
					"Browser has been idle for %v. Setting status to idle so new connections can be established",
					crm.conf.BrowserAutoSetIdleTimeoutInSecs,
				))
				crm.SetIdleOrStop()
				continue
			}

			shutdownCheck := crm.event.lastEventTimestamp.Add(crm.conf.BrowserAutoShutdownTimeoutInSecs)

			// browser is idle for BrowserAutoShutdownTimeoutInSecs seconds
			if now.After(shutdownCheck) {
				if crm.conf.EnableBrowserAutoShutdown {
					log.Debug().Ctx(crm.ea.ctx).Msg(fmt.Sprintf(
						"Browser has been idle for %v",
						crm.conf.BrowserAutoShutdownTimeoutInSecs,
					))
					crm.event.receiver <- EventData{
						BrowserID: crm.meta.browserID,
						EventType: ChromiumEventBrowserIdle,
					}
				} else {
					crm.event.idleMessageSync.Do(func() {
						log.Debug().Ctx(crm.ea.ctx).Msg(fmt.Sprintf(
							"Browser has been idle for over %v. Consider enabling %s",
							crm.conf.BrowserAutoShutdownTimeoutInSecs,
							config.ChromeEnableBrowserAutoShutdown,
						))
					})
				}
			}
		case <-crm.event.tickStopper:
			return
		}
	}
}

package chrome

import (
	"chromium-websocket-proxy/chromeprofile"
	"chromium-websocket-proxy/config"
	"chromium-websocket-proxy/logger"
	"context"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
	"strconv"
	"sync"
	"time"
)

type IChrome interface {
	Stop()
	Start() error
	DebugUrl() string
	Options() config.ChromeConfigOptions
	Config() config.ChromeConfig
	Ctx() context.Context
	BrowserID() uuid.UUID
	Port() int
	SessionId() uuid.UUID
	SetSessionId(uuid.UUID)
	IsIdle() bool
	IsNew() bool
	SetNotIdle()
	SetIdleOrStop()
	StartTicker()
	PauseTicker()
}

type Chrome struct {
	ea        ctxWithCancel
	ctx       context.Context
	cancel    context.CancelFunc
	sessionId uuid.UUID
	port      int
	conf      config.ChromeConfig
	mutex     sync.RWMutex
	meta      Meta
	event     event
	isIdle    bool
	isNew     bool
	isNewOnce sync.Once
	options   config.ChromeConfigOptions
}

type EventData struct {
	BrowserID uuid.UUID
	EventType EventType
}

type CreateChromePayload struct {
	Port          int
	SessionId     uuid.UUID
	EventReceiver chan EventData
	Options       config.ChromeConfigOptions
}

type ctxWithCancel struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type event struct {
	isPaused           bool
	ticker             *time.Ticker
	receiver           chan EventData
	tickStopper        chan bool
	idleMessageSync    sync.Once
	lastEventTimestamp time.Time
}

type EventType string

const (
	ChromiumEventTargetToDestroy  EventType = "ChromiumEventTargetToDestroy"
	ChromiumEventBrowserDestroyed EventType = "ChromiumEventBrowserDestroyed"
	ChromiumEventBrowserIdle      EventType = "ChromiumEventBrowserIdle"
)

func NewChrome(
	payload CreateChromePayload,
) IChrome {
	conf := config.Get().GetChromeConfig()

	// format chrome opts
	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("remote-debugging-port", strconv.Itoa(payload.Port)),
		chromedp.Flag("disable-extensions", true),
	)

	if conf.EnableCustomChromeProfiles && len(payload.Options.Profile) > 0 {
		profile, exists := chromeprofile.GetProfileByTag(payload.Options.Profile)
		if exists {
			opts = append(opts, chromedp.Flag("user-data-dir", chromeprofile.ProfilesDir))
			opts = append(opts, chromedp.Flag("profile-directory", profile))
		}
	}

	if !conf.Headless {
		opts = append(opts, chromedp.Flag("headless", conf.Headless))
	}

	crm := Chrome{
		port:      payload.Port,
		sessionId: payload.SessionId,
		options:   payload.Options,
		conf:      conf,
		event: event{
			isPaused:        true, // begin with true so that we don't start looping until it is started
			receiver:        payload.EventReceiver,
			tickStopper:     make(chan bool),
			idleMessageSync: sync.Once{},
		},
		isIdle:    true,
		isNew:     true,
		isNewOnce: sync.Once{},
	}

	// setup logger with ExecAllocator ctx
	crm.ea.ctx, crm.ea.cancel = chromedp.NewExecAllocator(context.Background(), opts...)

	if payload.SessionId != uuid.Nil {
		crm.ea.ctx = context.WithValue(crm.ea.ctx, logger.SessionIdTrackingKey, payload.SessionId)
	}
	if len(payload.Options.Profile) > 0 {
		crm.ea.ctx = context.WithValue(
			crm.ea.ctx,
			logger.BrowserProfileTrackingKey,
			payload.Options.Profile,
		)
	}

	// create chromedp ctx from exec allocator
	crm.ctx, crm.cancel = chromedp.NewContext(crm.ea.ctx)
	return &crm
}

func (crm *Chrome) Ctx() context.Context {
	return crm.ctx
}

func (crm *Chrome) DebugUrl() string {
	return crm.meta.debugUrl
}

func (crm *Chrome) SessionId() uuid.UUID {
	return crm.sessionId
}

func (crm *Chrome) SetSessionId(sessionId uuid.UUID) {
	crm.sessionId = sessionId
	crm.ea.ctx = context.WithValue(crm.ea.ctx, logger.SessionIdTrackingKey, sessionId)
}

func (crm *Chrome) Options() config.ChromeConfigOptions {
	return crm.options
}

func (crm *Chrome) Config() config.ChromeConfig {
	return crm.conf
}

func (crm *Chrome) FirstPageTargetID() target.ID {
	return crm.meta.firstPageTargetID
}

func (crm *Chrome) BrowserID() uuid.UUID {
	return crm.meta.browserID
}

func (crm *Chrome) Port() int {
	return crm.port
}

func (crm *Chrome) IsIdle() bool {
	return crm.isIdle
}

func (crm *Chrome) IsNew() bool {
	return crm.isNew
}

func (crm *Chrome) SetNotIdle() {
	crm.isIdle = false
	crm.isNewOnce.Do(func() {
		crm.isNew = false
	})
}

func (crm *Chrome) SetIdleOrStop() {
	log := logger.Get()
	if crm.conf.EnableBrowserReuse {
		crm.isIdle = true
		crm.SetSessionId(uuid.Nil)
		log.Info().Ctx(crm.ea.ctx).Msg("set chrome instance to idle for reuse")
	} else if crm.sessionId != uuid.Nil {
		crm.SetSessionId(uuid.Nil)
		crm.event.receiver <- EventData{
			BrowserID: crm.meta.browserID,
			EventType: ChromiumEventBrowserDestroyed,
		}
	}
}

func (crm *Chrome) Stop() {
	crm.PauseTicker()
	crm.cancel()
	crm.ea.cancel()
}

func (crm *Chrome) PauseTicker() {
	if !crm.event.isPaused {
		crm.event.isPaused = true
		crm.event.ticker.Stop()
		crm.event.tickStopper <- true
	}
}

func (crm *Chrome) StartTicker() {
	if crm.event.isPaused {
		crm.event.isPaused = false
		crm.event.ticker = time.NewTicker(500 * time.Millisecond)
		crm.event.lastEventTimestamp = time.Now()
		go crm.onTick()
	}
}

// Start is its own function for simplifying unit tests where chrome is mocked
func (crm *Chrome) Start() error {
	// ensure the first tab is created
	if err := chromedp.Run(crm.ctx); err != nil {
		return err
	}

	// set metadata from running instance
	if err := crm.fetchAndSetMeta(); err != nil {
		return err
	}

	// register browser listener
	chromedp.ListenBrowser(crm.ctx, crm.onBrowserEvent)

	// setup ticker after browser listener is registered
	crm.StartTicker()

	log := logger.Get()
	log.Debug().Ctx(crm.ea.ctx).Msg("started chrome instance")
	return nil
}

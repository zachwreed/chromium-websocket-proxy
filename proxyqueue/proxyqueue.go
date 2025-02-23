package proxyqueue

import (
	"chromium-websocket-proxy/chromepool"
	"chromium-websocket-proxy/config"
	"chromium-websocket-proxy/logger"
	"chromium-websocket-proxy/metrics"
	"chromium-websocket-proxy/websocketproxy"
	"container/list"
	"context"
	"errors"
	"github.com/google/uuid"
	"golang.org/x/time/rate"
	"net/http"
	"nhooyr.io/websocket"
	"strings"
	"sync"
	"time"
)

type ProxyQueue struct {
	queueTicker      *time.Ticker
	throughputTicker *time.Ticker
	list             *list.List
	listMux          sync.RWMutex
	tickStopC        chan bool
}

type ElementData struct {
	W                http.ResponseWriter
	R                *http.Request
	C                chan ProxyResult
	ChromeOptions    config.ChromeConfigOptions
	PriorityModifier float32
}

type ProxyResult string

const (
	Succeeded         ProxyResult = "Succeeded"
	ConnectionError   ProxyResult = "ConnectionError"
	SessionTimedOut   ProxyResult = "SessionTimedOut"
	UnableToGetChrome ProxyResult = "UnableToGetChrome" // retryable status
	Failed            ProxyResult = "Failed"
)

var once = sync.Once{}
var websocketAccept = websocket.Accept
var websocketDial = websocket.Dial
var newWebsocketProxy = websocketproxy.NewWebsocketProxy
var chromePoolGet = chromepool.Get

var pq *ProxyQueue

func Get() *ProxyQueue {
	once.Do(func() {
		pq = &ProxyQueue{
			list:             list.New(),
			tickStopC:        make(chan bool),
			queueTicker:      time.NewTicker(250 * time.Millisecond),
			throughputTicker: time.NewTicker(1000 * time.Millisecond),
		}
		go pq.onTick()
	})
	return pq
}

func Stop() {
	if pq != nil {
		pq.tickStopC <- true
	}
}

func NewElementData(
	w http.ResponseWriter,
	r *http.Request,
) (*ElementData, error) {
	cop := config.ChromeConfigOptionsPayload{
		Profile: r.URL.Query().Get("profile"),
	}

	co, err := config.NewCreateOptions(&cop)
	if err != nil {
		return nil, errors.New("unable to create options for chrome startup")
	}

	return &ElementData{
		W:             w,
		R:             r,
		C:             make(chan ProxyResult, 0),
		ChromeOptions: co,
	}, nil
}

func (pq *ProxyQueue) AddToList(el *ElementData) *list.Element {
	metrics.Get().InMemory.IncCounter(metrics.ProxyQueue, float32(1))
	metrics.Get().Remote.IncCounter(metrics.ProxyQueue, float32(1))

	pq.listMux.Lock()
	defer pq.listMux.Unlock()
	return pq.list.PushBack(el)
}

func (pq *ProxyQueue) RemoveFromList(el *list.Element) {
	metrics.Get().InMemory.IncCounter(metrics.ProxyQueue, float32(-1))
	metrics.Get().Remote.IncCounter(metrics.ProxyQueue, float32(-1))
	pq.listMux.Lock()
	defer pq.listMux.Unlock()
	pq.list.Remove(el)
}

func (pq *ProxyQueue) onTick() {
	log := logger.Get()
	cp := chromePoolGet()
	conf := config.Get()
	m := metrics.Get()

	for {
		select {
		case <-pq.queueTicker.C:
			pq.listMux.RLock()
			lLen := pq.list.Len()
			pq.listMux.RUnlock()

			if lLen == 0 || !cp.HasIdleChromeInstance() {
				continue
			}

			go func() {
				pq.listMux.Lock()
				el := pq.list.Front()
				if el == nil {
					pq.listMux.Unlock()
					return
				}
				pqe := pq.list.Remove(el).(*ElementData)
				pq.listMux.Unlock()
				if pqe == nil {
					return
				}

				log.Info().Ctx(pqe.R.Context()).Msg("attempting proxy session")
				res := pqe.proxy()

				// add back to list
				if res == UnableToGetChrome {
					pq.listMux.Lock()

					front := pq.list.Front()
					if front != nil {
						pq.list.InsertAfter(pqe, front)
						log.Info().Ctx(pqe.R.Context()).Msg("pushing session to after")
					} else {
						pq.list.PushFront(pqe)
					}
					pq.listMux.Unlock()
				} else {
					pqe.C <- res
				}
			}()
		case <-pq.throughputTicker.C:
			// do not calculate throughput for scale-up if at capacity
			go func() {
				if cp.IsPoolAtCapacity() {
					return
				}

				if pq.list.Len() == 0 {
					return
				}

				// Formula and Calculation of Throughput
				// can be calculated using the following formula:
				//
				// tp = qp/apt
				// where:
				// qp = queued proxy session / chrome instances
				// apt = average proxy session time to complete
				// tp = throughput

				// Get current mean count of in queue & working
				cag, err := m.InMemory.GetLastCounterAggregate(metrics.ProxyQueue)
				if err != nil {
					log.Err(err).Msg("GetLastCounterAggregate")
				}
				if cag == nil || cag.Count == 0 {
					return
				}

				// Get current mean time to complete a proxy
				sag, _ := m.InMemory.GetLastSampleAggregate(metrics.ProxyTimeSecs)

				// sag may be nil if we just received a load after a period of inactivity.
				// In this case, set apt equal to a config value
				var apt float64
				if sag == nil {
					apt = float64(25)
				} else {
					apt = sag.Mean()
				}

				l := float64(cp.GetInstancePoolLen())
				qp := float64(cag.Count) / l
				tp := qp / apt

				if tp <= conf.GetProxyQueueConfig().ThroughputScaleUpThreshold {
					return
				}
				log.Info().Float64("throughput", tp).Msg("scaling up chrome pool")

				err = cp.CreateNewInstance(conf.GetChromeConfig().DefaultOptions)
				if err != nil {
					log.Err(err).Msg("error scaling up")
				}
			}()
		case <-pq.tickStopC:
			return
		}
	}
}

func (pqe *ElementData) proxy() ProxyResult {
	log := logger.Get()

	crm, err := chromePoolGet().GetAvailableChrome(
		pqe.R.Context().Value(logger.SessionIdTrackingKey).(uuid.UUID),
		pqe.ChromeOptions,
	)
	if err != nil {
		log.Warn().Err(err).Ctx(pqe.R.Context()).Msg("unable to GetAvailableChrome")
		return UnableToGetChrome
	}
	defer (*crm).SetIdleOrStop()

	chromeCtx, cancel := context.WithCancel(pqe.R.Context())
	defer cancel()

	// dial chrome after getting instance
	chromeConn, _, err := websocketDial(chromeCtx, (*crm).DebugUrl(), nil)
	if err != nil {
		log.Error().Err(err).Ctx(pqe.R.Context()).Msg("unable to connect to chrome ws port")
		return ConnectionError
	}
	chromeConn.SetReadLimit(-1)
	defer chromeConn.CloseNow()

	// accept websocket after chrome is ready
	clientConn, err := websocketAccept(pqe.W, pqe.R, nil)
	if err != nil {
		log.Error().Ctx(pqe.R.Context()).Msg("unable to accept client connection")
		return ConnectionError
	}
	clientConn.SetReadLimit(-1)
	defer clientConn.CloseNow()

	limiter := rate.NewLimiter(rate.Every(time.Millisecond*10), 10)

	clientWs := newWebsocketProxy(
		clientConn,
		pqe.R.Context(),
		websocketproxy.Client,
		limiter,
		10,
	)

	chromeWs := newWebsocketProxy(
		chromeConn,
		chromeCtx,
		websocketproxy.Chrome,
		limiter,
		10,
	)

	chromeWs.SetWriteConnection(clientConn, pqe.R.Context())
	clientWs.SetWriteConnection(chromeConn, chromeCtx)

	errC := make(chan error, 0)

	start := time.Now()

	proxyLoop := func(wp *websocketproxy.WebsocketProxy) {
		for {
			err = wp.Proxy()
			if err != nil {
				errC <- err
			}
		}
	}

	go proxyLoop(clientWs)
	go proxyLoop(chromeWs)

	log.Info().Ctx(pqe.R.Context()).Msg("proxy client and chrome connections initialized")

	// block until error channel is received
	err = <-errC

	diff := time.Now().Sub(start)
	metrics.Get().InMemory.AddSample(metrics.ProxyTimeSecs, float32(diff.Seconds()))
	metrics.Get().Remote.AddSample(metrics.ProxyTimeSecs, float32(diff.Seconds()))

	if err == nil ||
		websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
		websocket.CloseStatus(err) == websocket.StatusGoingAway ||
		websocket.CloseStatus(err) == websocket.StatusNoStatusRcvd {
		return Succeeded
	}

	if strings.Contains(err.Error(), "use of closed network connection") {
		return Succeeded
	}

	if errors.Is(err, context.Canceled) {
		log.Error().Ctx(pqe.R.Context()).Msg("session timed out")
		return SessionTimedOut
	}

	return Failed
}

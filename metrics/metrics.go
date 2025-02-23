package metrics

import (
	"chromium-websocket-proxy/config"
	"github.com/hashicorp/go-metrics"
	"sync"
	"time"
)

type Metrics struct {
	InMemory InMemory
	Remote   Remote
}

type MetricKey string

const (
	ProxyQueue      MetricKey = "proxy-queue"
	ProxyTimeSecs   MetricKey = "proxy-time-secs"
	ChromeInstances MetricKey = "chrome-instances"
)

var once = sync.Once{}

var m *Metrics

func isStringPopulated(val string) bool {
	return len(val) > 0
}

func Init() (err error) {
	once.Do(func() {
		conf := config.Get().GetMetricsConfig()

		var rs metrics.MetricSink
		var rsm *metrics.Metrics
		if isStringPopulated(conf.StatsiteSink) {
			rs, err = metrics.NewStatsiteSink(conf.StatsiteSink)
		} else if isStringPopulated(conf.StatsDSink) {
			rs, err = metrics.NewStatsdSink(conf.StatsDSink)
		}
		if err != nil {
			return
		}

		var imm *metrics.Metrics
		ims := metrics.NewInmemSink(5*time.Minute, 15*time.Minute)

		imm, err = metrics.New(
			metrics.DefaultConfig("chromium-websocket-proxy-in-memory"),
			ims,
		)
		if err != nil {
			return
		}

		if rs != nil {
			rsm, err = metrics.New(
				metrics.DefaultConfig("chromium-websocket-proxy"),
				rs,
			)
			if err != nil {
				return
			}
		}

		m = &Metrics{
			InMemory: InMemory{
				sink:    ims,
				metrics: imm,
			},
			Remote: Remote{
				sink:    rs,
				metrics: rsm,
			},
		}
	})
	return err
}

func Get() *Metrics {
	return m
}

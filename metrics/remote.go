package metrics

import (
	"github.com/hashicorp/go-metrics"
)

type Remote struct {
	sink    metrics.MetricSink
	metrics *metrics.Metrics
}

func (r *Remote) AddSample(key MetricKey, val float32) {
	if r.metrics != nil {
		r.metrics.AddSample([]string{string(key)}, val)
	}
}

func (r *Remote) IncCounter(key MetricKey, val float32) {
	if r.metrics != nil {
		r.metrics.IncrCounter([]string{string(key)}, val)
	}
}

func (r *Remote) SetGauge(key MetricKey, val float32) {
	if r.metrics != nil {
		r.metrics.SetGauge([]string{string(key)}, val)
	}
}

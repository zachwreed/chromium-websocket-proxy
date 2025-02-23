package metrics

import (
	"errors"
	"github.com/hashicorp/go-metrics"
)

type InMemory struct {
	sink    *metrics.InmemSink
	metrics *metrics.Metrics
}

func (im *InMemory) AddSample(key MetricKey, val float32) {
	im.sink.AddSample([]string{string(key)}, val)
}

func (im *InMemory) GetLastSampleAggregate(key MetricKey) (*metrics.AggregateSample, error) {
	ims := im.sink.Data()
	if len(ims) == 0 {
		return nil, errors.New("no metrics available")
	}
	c := ims[len(ims)-1].Samples[string(key)]
	return c.AggregateSample, nil
}

func (im *InMemory) IncCounter(key MetricKey, val float32) {
	im.sink.IncrCounter([]string{string(key)}, val)
}

func (im *InMemory) GetLastCounterAggregate(key MetricKey) (*metrics.AggregateSample, error) {
	ims := im.sink.Data()
	if len(ims) == 0 {
		return nil, errors.New("no metrics available")
	}
	c := ims[len(ims)-1].Counters[string(key)]
	return c.AggregateSample, nil
}

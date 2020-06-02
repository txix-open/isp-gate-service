package service

import (
	"github.com/integration-system/isp-lib/v2/metric"
	"github.com/rcrowley/go-metrics"
	"sync"
	"time"
)

const (
	defaultSampleSize = 2048
)

var Metrics = &metricService{mh: nil}

type (
	metricService struct {
		mh *metricHolder
	}

	metricHolder struct {
		methodHistograms map[string]metrics.Histogram
		methodLock       sync.RWMutex
		statusCounters   map[string]metrics.Counter
		statusLock       sync.RWMutex
	}
)

func (m *metricService) Init() {
	m.mh = &metricHolder{
		methodHistograms: make(map[string]metrics.Histogram),
		statusCounters:   make(map[string]metrics.Counter),
	}
}

func (m *metricService) UpdateMethodResponseTime(uri string, time time.Duration) {
	if m.notEmptyHolder() {
		m.getOrRegisterHistogram(uri).Update(int64(time))
	}
}

func (m *metricService) UpdateStatusCounter(status string) {
	if m.notEmptyHolder() {
		m.getOrRegisterCounter(status).Inc(1)
	}
}

func (m *metricService) getOrRegisterHistogram(uri string) metrics.Histogram {
	m.mh.methodLock.RLock()
	histogram, ok := m.mh.methodHistograms[uri]
	m.mh.methodLock.RUnlock()
	if ok {
		return histogram
	}

	m.mh.methodLock.Lock()
	defer m.mh.methodLock.Unlock()
	if d, ok := m.mh.methodHistograms[uri]; ok {
		return d
	}
	histogram = metrics.GetOrRegisterHistogram(
		"http.response.time_"+uri,
		metric.GetRegistry(),
		metrics.NewUniformSample(defaultSampleSize),
	)
	m.mh.methodHistograms[uri] = histogram
	return histogram
}

func (m *metricService) getOrRegisterCounter(status string) metrics.Counter {
	m.mh.statusLock.RLock()
	d, ok := m.mh.statusCounters[status]
	m.mh.statusLock.RUnlock()
	if ok {
		return d
	}

	m.mh.statusLock.Lock()
	defer m.mh.statusLock.Unlock()
	if d, ok := m.mh.statusCounters[status]; ok {
		return d
	}
	d = metrics.GetOrRegisterCounter("http.response.count."+status, metric.GetRegistry())
	m.mh.statusCounters[status] = d
	return d
}

func (m *metricService) notEmptyHolder() bool {
	return m.mh != nil
}

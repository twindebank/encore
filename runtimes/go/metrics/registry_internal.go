package metrics

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"

	"encore.dev/appruntime/shared/nativehist"
	"encore.dev/appruntime/shared/reqtrack"
)

type Registry struct {
	rt       *reqtrack.RequestTracker
	numSvcs  uint16
	tsid     uint64
	registry sync.Map // map[registryKey]*timeseries

	serviceLabels sync.Map // map[string][]KeyValue — extra labels per service name
}

func NewRegistry(rt *reqtrack.RequestTracker, numServicesInBinary int) *Registry {
	return &Registry{rt: rt, numSvcs: uint16(numServicesInBinary)}
}

func (r *Registry) Collect() []CollectedMetric {
	metrics := make([]CollectedMetric, 0, 128)
	r.registry.Range(func(key, value any) bool {
		switch val := value.(type) {
		case *timeseries[int64]:
			metrics = append(metrics, CollectedMetric{
				Info:         val.info,
				TimeSeriesID: val.id,
				Labels:       val.labels,
				Val:          val.value,
				Valid:        val.valid,
			})
		case *timeseries[uint64]:
			metrics = append(metrics, CollectedMetric{
				Info:         val.info,
				TimeSeriesID: val.id,
				Labels:       val.labels,
				Val:          val.value,
				Valid:        val.valid,
			})
		case *timeseries[float64]:
			metrics = append(metrics, CollectedMetric{
				Info:         val.info,
				TimeSeriesID: val.id,
				Labels:       val.labels,
				Val:          val.value,
				Valid:        val.valid,
			})
		case *timeseries[*nativehist.Histogram]:
			metrics = append(metrics, CollectedMetric{
				Info:         val.info,
				TimeSeriesID: val.id,
				Labels:       val.labels,
				Val:          val.value,
				Valid:        val.valid,
			})
		default:
			panic(fmt.Sprintf("unhandled timeseries type %T", val))
		}
		return true
	})
	return metrics
}

type MetricType int

const (
	CounterType MetricType = iota
	GaugeType
	HistogramType
)

type MetricInfo interface {
	Name() string
	Type() MetricType
	SvcNum() uint16
}

type CollectedMetric struct {
	Info         MetricInfo
	TimeSeriesID uint64
	Labels       []KeyValue
	Val          any // []T where T is any of Value
	Valid        []atomic.Bool

	// ServiceLabels maps service names to additional labels that should be
	// included when exporting this metric for that service. It is nil when
	// no service labels have been registered.
	ServiceLabels map[string][]KeyValue
}

type registryKey struct {
	metricName string
	labels     any // guaranteed to be comparable
}

type timeseries[T any] struct {
	info   MetricInfo
	id     uint64
	init   initGate
	labels []KeyValue
	value  []T
	valid  []atomic.Bool
}

func (ts *timeseries[V]) setup(labels []KeyValue) {
	ts.init.Start()
	defer ts.init.Done()
	ts.labels = labels
}

type KeyValue struct {
	Key   string
	Value string
}

func getTS[T any](r *Registry, name string, labels any, info MetricInfo) (ts *timeseries[T], loaded bool) {
	key := registryKey{metricName: name, labels: labels}
	if val, ok := r.registry.Load(key); ok {
		return val.(*timeseries[T]), true
	}
	val, loaded := r.registry.LoadOrStore(key, &timeseries[T]{
		info: info,
		id:   atomic.AddUint64(&r.tsid, 1),
	})
	return val.(*timeseries[T]), loaded
}

// RegisterServiceLabels registers additional labels to be included with all
// metrics exported for the named service. This is useful for enriching built-in
// metrics (like e_requests_total) with custom metadata for alert routing or
// dashboard filtering.
//
// Subsequent calls for the same service replace previous labels.
func (r *Registry) RegisterServiceLabels(serviceName string, labels map[string]string) {
	kvs := make([]KeyValue, 0, len(labels))
	for k, v := range labels {
		kvs = append(kvs, KeyValue{Key: k, Value: v})
	}
	sort.Slice(kvs, func(i, j int) bool { return kvs[i].Key < kvs[j].Key })
	r.serviceLabels.Store(serviceName, kvs)
}

// ServiceLabels returns a snapshot of all registered service labels.
func (r *Registry) ServiceLabels() map[string][]KeyValue {
	result := make(map[string][]KeyValue)
	r.serviceLabels.Range(func(key, value any) bool {
		result[key.(string)] = value.([]KeyValue)
		return true
	})
	if len(result) == 0 {
		return nil
	}
	return result
}

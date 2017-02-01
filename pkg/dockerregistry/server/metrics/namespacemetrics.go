package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespaceLabelName = "name"
)

var (
	namespaceMetricVecSync sync.RWMutex
	globalMetricVec        map[metricFQName]*metricVec
	namespaceMetricVec     map[string]metricVecMap
)

type metricFQName string
type metricVecMap map[metricFQName]*metricVec
type constructorMetricVec func(opts MetricVecOpts) *metricVec

type metricVec struct {
	summaryVec *prometheus.SummaryVec
	counterVec *prometheus.CounterVec
}

type MetricVecOpts struct {
	// Namespace, Subsystem, and Name are components of the fully-qualified
	// name of the Metric (created by joining these components with
	// "_"). Only Name is mandatory, the others merely help structuring the
	// name. Note that the fully-qualified name of the metric must be a
	// valid Prometheus metric name.
	Namespace string
	Subsystem string
	Name      string

	// Help provides information about this metric. Mandatory!
	// Metrics with the same fully-qualified name must have the same Help
	// string.
	Help string

	LabelNames []string

	Constructor constructorMetricVec
}

func (m MetricVecOpts) String() string {
	return prometheus.BuildFQName(m.Namespace, m.Subsystem, m.Name)
}

func MakeSummaryVec(opts MetricVecOpts) *metricVec {
	return &metricVec{
		summaryVec: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace: opts.Namespace,
				Subsystem: opts.Subsystem,
				Name:      opts.Name,
				Help:      opts.Help,
			},
			opts.LabelNames,
		),
	}
}

func RegisterNSMetricVec(namespace string, opts MetricVecOpts) *metricVec {
	namespaceMetricVecSync.Lock()
	defer namespaceMetricVecSync.Unlock()

	metricName := metricFQName(opts.String())

	if namespaceMetricVec == nil {
		namespaceMetricVec = make(map[string]metricVecMap)
		globalMetricVec = make(map[metricFQName]*metricVec)
	}

	if _, ok := namespaceMetricVec[namespace]; !ok {
		namespaceMetricVec[namespace] = make(metricVecMap)
	}

	vec, ok := namespaceMetricVec[namespace][metricName]
	if ok {
		return vec
	}

	vec, ok = globalMetricVec[metricName]
	if ok {
		namespaceMetricVec[namespace][metricName] = vec
		return vec
	}

	vec = opts.Constructor(opts)

	switch {
	case vec.summaryVec != nil:
		prometheus.MustRegister(vec.summaryVec)
	case vec.counterVec != nil:
		prometheus.MustRegister(vec.counterVec)
	default:
		panic("unexpectecd metric")
	}

	globalMetricVec[metricName] = vec
	namespaceMetricVec[namespace][metricName] = vec

	return vec
}

func NewNSTimer(namespace string, opts MetricVecOpts, labelValues []string) *metricTimer {
	vec := RegisterNSMetricVec(namespace, opts)

	if vec.summaryVec == nil {
		panic("metric can't be used as timer")
	}

	return &metricTimer{
		collector: vec.summaryVec,
		labels:    labelValues,
		startTime: time.Now(),
	}
}

type metricTimer struct {
	collector *prometheus.SummaryVec
	labels    []string
	startTime time.Time
}

func (m *metricTimer) Stop() {
	m.collector.WithLabelValues(m.labels...).Observe(float64(time.Since(m.startTime) / time.Microsecond))
}

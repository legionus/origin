package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	Register()
}

const (
	registryNamespace = "openshift"
	registrySubsystem = "registry"
)

var (
	RequestDurationSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: registryNamespace,
			Subsystem: registrySubsystem,
			Name:      "request_duration_microseconds",
			Help:      "Request latency summary in microseconds for each operation",
		},
		[]string{"operation", "name"},
	)
)

var registerMetrics sync.Once

// Register all metrics.
func Register() {
	// Register the metrics.
	registerMetrics.Do(func() {
		prometheus.MustRegister(RequestDurationSummary)
	})
}

func IncCounterVec(collector *prometheus.CounterVec, labels []string) {
	collector.WithLabelValues(labels...).Inc()
}

func NewTimer(collector *prometheus.SummaryVec, labels []string) *metricTimer {
	return &metricTimer{
		collector: collector,
		labels:    labels,
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

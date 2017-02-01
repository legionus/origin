package metrics

const (
	registryNamespace = "openshift"
	registrySubsystem = "registry"
)

var (
	RequestDurationSummaryName = MetricVecOpts{
		Namespace:   registryNamespace,
		Subsystem:   registrySubsystem,
		Name:        "request_duration_microseconds",
		Help:        "Request latency summary in microseconds for each operation",
		LabelNames:  []string{"operation", "name"},
		Constructor: MakeSummaryVec,
	}
)

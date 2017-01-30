package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

func Handler(handler http.Handler) http.Handler {
	return prometheus.InstrumentHandler(registryNamespace, handler)
}

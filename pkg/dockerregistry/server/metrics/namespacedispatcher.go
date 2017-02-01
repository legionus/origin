package metrics

import (
	"sync"
	"net/http"

	"github.com/docker/distribution/context"
	"github.com/docker/distribution/registry/handlers"

	gorillahandlers "github.com/gorilla/handlers"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

const (
	// Capacity for the channel to collect metrics and descriptors.
	capMetricChan = 1000
)

type namespaceHandler struct {
	ctx  *handlers.Context
	name string
}

func NamespaceDispatcher(ctx *handlers.Context, r *http.Request) http.Handler {
	handler := &namespaceHandler{
		ctx:  ctx,
		name: context.GetStringValue(ctx, "vars.name"),
	}

	return gorillahandlers.MethodHandler{
		"GET": http.HandlerFunc(handler.Get),
	}
}

func (h *namespaceHandler) Get(w http.ResponseWriter, req *http.Request) {
	// format := "prometheus" // json

	if len(h.name) == 0 {
		return
	}

	namespaceMetricVecSync.RLock()
	defer namespaceMetricVecSync.RUnlock()

	namespaceMetrics, ok := namespaceMetricVec[h.name]
	if !ok {
		return // Not found ?
	}

	metricChan := make(chan prometheus.Metric, capMetricChan)

	// Drain metricChan in case of premature return.
	defer func() {
		for _ = range metricChan {
		}
	}()

	wg := sync.WaitGroup{}

	for _, vec := range namespaceMetrics {
		switch {
		case vec.summaryVec != nil:
			go func(mv *metricVec) {
				defer wg.Done()
				mv.summaryVec.Collect(metricChan)
			}(vec)
		case vec.counterVec != nil:
			go func(mv *metricVec) {
				defer wg.Done()
				mv.counterVec.Collect(metricChan)
			}(vec)
		}
	}

	go func() {
		wg.Wait()
		close(metricChan)
	}()

	// Gather.
	for metric := range metricChan {
		m := &dto.Metric{}
		if err := metric.Write(m); err != nil {
			panic(err)
		}
		found := false
		for _, lbl := range m.Label {
			if lbl.GetName() == namespaceLabelName && lbl.GetValue() == h.name {
				found = true
				break
			}
		}
		if found {
			w.Write([]byte(m.String()))
		}
	}
}

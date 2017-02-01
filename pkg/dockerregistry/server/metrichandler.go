package server

import (
	"net/http"

	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/auth"
	"github.com/docker/distribution/registry/handlers"

	"github.com/openshift/origin/pkg/dockerregistry/server/metrics"
)

func RegisterMetricHandler(app *handlers.App) {
	emptyAccess := func(r *http.Request) []auth.Access {
		return nil
	}
	extensionsRouter := app.NewRoute().PathPrefix("/extensions/v2/").Subrouter()
	app.RegisterRoute(
		extensionsRouter.Path("/metrics").Methods("GET"),
		metrics.Dispatcher,
		handlers.NameNotRequired,
		emptyAccess,
	)
	app.RegisterRoute(
		extensionsRouter.Path("/{name:"+reference.NameRegexp.String()+"}/metrics").Methods("GET"),
		metrics.Dispatcher,
		handlers.NameRequired,
		emptyAccess,
	)
}

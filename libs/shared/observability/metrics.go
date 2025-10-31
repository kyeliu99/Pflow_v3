package observability

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RegisterMetricsEndpoint exposes Prometheus metrics on /metrics.
func RegisterMetricsEndpoint(router chi.Router) {
	router.Method(http.MethodGet, "/metrics", promhttp.Handler())
}

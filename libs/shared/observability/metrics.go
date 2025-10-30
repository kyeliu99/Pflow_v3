package observability

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RegisterMetricsEndpoint exposes Prometheus metrics on /metrics.
func RegisterMetricsEndpoint(router *gin.Engine) {
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
}

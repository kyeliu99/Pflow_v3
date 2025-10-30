package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/pflow/shared/config"
	"github.com/pflow/shared/httpx"
	"github.com/pflow/shared/observability"
)

func main() {
	cfg := config.Load()

	server := httpx.New()
	observability.RegisterMetricsEndpoint(server.Engine)

	server.Engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": cfg.ServiceName})
	})

	// Proxy endpoints to domain services (stub implementation for now)
	server.Engine.Any("/api/*proxyPath", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "gateway placeholder",
			"path":    c.Param("proxyPath"),
		})
	})

	addr := fmt.Sprintf(":%s", cfg.HTTPPort)
	log.Printf("gateway listening on %s", addr)
	if err := server.Start(addr); err != nil {
		log.Fatalf("gateway stopped: %v", err)
	}
}

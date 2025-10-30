package httpx

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Server wraps a Gin engine with graceful shutdown helpers.
type Server struct {
	Engine     *gin.Engine
	httpServer *http.Server
}

// New creates a new HTTP server with sane defaults.
func New() *Server {
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(gin.Logger())

	return &Server{Engine: engine}
}

// Start begins serving HTTP traffic on the provided address.
func (s *Server) Start(addr string) error {
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.Engine,
	}
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

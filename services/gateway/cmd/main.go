package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pflow/shared/config"
	"github.com/pflow/shared/httpx"
	"github.com/pflow/shared/observability"
)

type gateway struct {
	client       *http.Client
	serviceName  string
	formBase     string
	identityBase string
	ticketBase   string
	workflowBase string
}

func newGateway(cfg *config.AppConfig) *gateway {
	return &gateway{
		client:       &http.Client{Timeout: 10 * time.Second},
		serviceName:  cfg.ServiceName,
		formBase:     trimTrailingSlash(cfg.FormServiceURL),
		identityBase: trimTrailingSlash(cfg.IdentityServiceURL),
		ticketBase:   trimTrailingSlash(cfg.TicketServiceURL),
		workflowBase: trimTrailingSlash(cfg.WorkflowServiceURL),
	}
}

func main() {
	cfg := config.Load()
	gw := newGateway(cfg)

	server := httpx.New()
	observability.RegisterMetricsEndpoint(server.Engine)

	server.Engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": gw.serviceName})
	})

	api := server.Engine.Group("/api")
	gw.registerRoutes(api)

	port := cfg.ResolveHTTPPort("8080")
	addr := fmt.Sprintf(":%s", port)
	log.Printf("gateway listening on %s", addr)

	if err := server.Start(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("gateway stopped: %v", err)
	}
}

func (g *gateway) registerRoutes(router gin.IRouter) {
	router.GET("/forms", g.proxy(http.MethodGet, func(c *gin.Context) string {
		return g.formBase + "/forms"
	}))
	router.POST("/forms", g.proxy(http.MethodPost, func(c *gin.Context) string {
		return g.formBase + "/forms"
	}))
	router.GET("/forms/:id", g.proxy(http.MethodGet, func(c *gin.Context) string {
		return g.formBase + "/forms/" + c.Param("id")
	}))
	router.PUT("/forms/:id", g.proxy(http.MethodPut, func(c *gin.Context) string {
		return g.formBase + "/forms/" + c.Param("id")
	}))
	router.DELETE("/forms/:id", g.proxy(http.MethodDelete, func(c *gin.Context) string {
		return g.formBase + "/forms/" + c.Param("id")
	}))

	router.GET("/tickets", g.proxy(http.MethodGet, func(c *gin.Context) string {
		return g.ticketBase + "/tickets"
	}))
	router.POST("/tickets", g.proxy(http.MethodPost, func(c *gin.Context) string {
		return g.ticketBase + "/tickets"
	}))
	router.GET("/tickets/:id", g.proxy(http.MethodGet, func(c *gin.Context) string {
		return g.ticketBase + "/tickets/" + c.Param("id")
	}))
	router.PATCH("/tickets/:id", g.proxy(http.MethodPatch, func(c *gin.Context) string {
		return g.ticketBase + "/tickets/" + c.Param("id")
	}))
	router.DELETE("/tickets/:id", g.proxy(http.MethodDelete, func(c *gin.Context) string {
		return g.ticketBase + "/tickets/" + c.Param("id")
	}))
	router.POST("/tickets/:id/resolve", g.proxy(http.MethodPost, func(c *gin.Context) string {
		return g.ticketBase + "/tickets/" + c.Param("id") + "/resolve"
	}))

	router.GET("/users", g.proxy(http.MethodGet, func(c *gin.Context) string {
		return g.identityBase + "/identity/users"
	}))
	router.POST("/users", g.proxy(http.MethodPost, func(c *gin.Context) string {
		return g.identityBase + "/identity/users"
	}))
	router.GET("/users/:id", g.proxy(http.MethodGet, func(c *gin.Context) string {
		return g.identityBase + "/identity/users/" + c.Param("id")
	}))
	router.PUT("/users/:id", g.proxy(http.MethodPut, func(c *gin.Context) string {
		return g.identityBase + "/identity/users/" + c.Param("id")
	}))
	router.DELETE("/users/:id", g.proxy(http.MethodDelete, func(c *gin.Context) string {
		return g.identityBase + "/identity/users/" + c.Param("id")
	}))

	router.GET("/workflows", g.proxy(http.MethodGet, func(c *gin.Context) string {
		return g.workflowBase + "/workflows"
	}))
	router.POST("/workflows", g.proxy(http.MethodPost, func(c *gin.Context) string {
		return g.workflowBase + "/workflows"
	}))
	router.GET("/workflows/:id", g.proxy(http.MethodGet, func(c *gin.Context) string {
		return g.workflowBase + "/workflows/" + c.Param("id")
	}))
	router.PUT("/workflows/:id", g.proxy(http.MethodPut, func(c *gin.Context) string {
		return g.workflowBase + "/workflows/" + c.Param("id")
	}))
	router.DELETE("/workflows/:id", g.proxy(http.MethodDelete, func(c *gin.Context) string {
		return g.workflowBase + "/workflows/" + c.Param("id")
	}))
	router.POST("/workflows/:id/publish", g.proxy(http.MethodPost, func(c *gin.Context) string {
		return g.workflowBase + "/workflows/" + c.Param("id") + "/publish"
	}))

	router.GET("/overview", g.overviewHandler)
}

func (g *gateway) proxy(method string, target func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		g.forward(c, method, target(c))
	}
}

func (g *gateway) forward(c *gin.Context, method, target string) {
	var body io.Reader
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		rawBody, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
			return
		}
		body = bytes.NewReader(rawBody)
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), method, target, body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to build upstream request: %v", err)})
		return
	}

	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		if contentType := c.GetHeader("Content-Type"); contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
	}
	req.URL.RawQuery = c.Request.URL.RawQuery

	resp, err := g.client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("upstream request failed: %v", err)})
		return
	}
	defer resp.Body.Close()

	for key := range c.Writer.Header() {
		c.Writer.Header().Del(key)
	}
	for key, values := range resp.Header {
		if strings.EqualFold(key, "Content-Length") {
			continue
		}
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}

	c.Status(resp.StatusCode)
	if resp.StatusCode == http.StatusNoContent {
		return
	}
	if _, err := io.Copy(c.Writer, resp.Body); err != nil {
		log.Printf("gateway: failed to copy response body: %v", err)
	}
}

func (g *gateway) overviewHandler(c *gin.Context) {
	ctx := c.Request.Context()

	forms, err := g.fetchList(ctx, g.formBase+"/forms")
	if err != nil {
		g.renderUpstreamError(c, err)
		return
	}

	tickets, err := g.fetchList(ctx, g.ticketBase+"/tickets")
	if err != nil {
		g.renderUpstreamError(c, err)
		return
	}

	users, err := g.fetchList(ctx, g.identityBase+"/identity/users")
	if err != nil {
		g.renderUpstreamError(c, err)
		return
	}

	workflows, err := g.fetchList(ctx, g.workflowBase+"/workflows")
	if err != nil {
		g.renderUpstreamError(c, err)
		return
	}

	ticketStatus := map[string]int{}
	for _, ticket := range tickets {
		if status, ok := ticket["status"].(string); ok {
			ticketStatus[status]++
		}
	}

	publishedWorkflows := 0
	for _, wf := range workflows {
		if published, ok := wf["published"].(bool); ok && published {
			publishedWorkflows++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"forms": gin.H{
				"total": len(forms),
			},
			"tickets": gin.H{
				"total":    len(tickets),
				"byStatus": ticketStatus,
			},
			"users": gin.H{
				"total": len(users),
			},
			"workflows": gin.H{
				"total":     len(workflows),
				"published": publishedWorkflows,
			},
		},
	})
}

func (g *gateway) fetchList(ctx context.Context, target string) ([]map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("upstream %s responded with %d: %s", target, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload.Data, nil
}

func (g *gateway) renderUpstreamError(c *gin.Context, err error) {
	log.Printf("gateway: overview failed: %v", err)
	c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
}

func trimTrailingSlash(value string) string {
	return strings.TrimRight(value, "/")
}

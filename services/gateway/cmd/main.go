package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

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
	observability.RegisterMetricsEndpoint(server.Router)

	server.Router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]any{"status": "ok", "service": gw.serviceName})
	})

	server.Router.Route("/api", gw.registerRoutes)

	port := cfg.ResolveServiceHTTPPort("gateway", "8080")
	addr := fmt.Sprintf(":%s", port)
	log.Printf("gateway listening on %s", addr)

	if err := server.Start(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("gateway stopped: %v", err)
	}
}

func (g *gateway) registerRoutes(router chi.Router) {
	router.Get("/forms", g.proxy(http.MethodGet, func(r *http.Request) string {
		return g.formBase + "/forms"
	}))
	router.Post("/forms", g.proxy(http.MethodPost, func(r *http.Request) string {
		return g.formBase + "/forms"
	}))
	router.Get("/forms/{id}", g.proxy(http.MethodGet, func(r *http.Request) string {
		return g.formBase + "/forms/" + chi.URLParam(r, "id")
	}))
	router.Put("/forms/{id}", g.proxy(http.MethodPut, func(r *http.Request) string {
		return g.formBase + "/forms/" + chi.URLParam(r, "id")
	}))
	router.Delete("/forms/{id}", g.proxy(http.MethodDelete, func(r *http.Request) string {
		return g.formBase + "/forms/" + chi.URLParam(r, "id")
	}))

	router.Get("/tickets", g.proxy(http.MethodGet, func(r *http.Request) string {
		return g.ticketBase + "/tickets"
	}))
	router.Post("/tickets", g.proxy(http.MethodPost, func(r *http.Request) string {
		return g.ticketBase + "/tickets"
	}))
	router.Get("/tickets/{id}", g.proxy(http.MethodGet, func(r *http.Request) string {
		return g.ticketBase + "/tickets/" + chi.URLParam(r, "id")
	}))
	router.Patch("/tickets/{id}", g.proxy(http.MethodPatch, func(r *http.Request) string {
		return g.ticketBase + "/tickets/" + chi.URLParam(r, "id")
	}))
	router.Delete("/tickets/{id}", g.proxy(http.MethodDelete, func(r *http.Request) string {
		return g.ticketBase + "/tickets/" + chi.URLParam(r, "id")
	}))
	router.Post("/tickets/{id}/resolve", g.proxy(http.MethodPost, func(r *http.Request) string {
		return g.ticketBase + "/tickets/" + chi.URLParam(r, "id") + "/resolve"
	}))

	router.Get("/users", g.proxy(http.MethodGet, func(r *http.Request) string {
		return g.identityBase + "/identity/users"
	}))
	router.Post("/users", g.proxy(http.MethodPost, func(r *http.Request) string {
		return g.identityBase + "/identity/users"
	}))
	router.Get("/users/{id}", g.proxy(http.MethodGet, func(r *http.Request) string {
		return g.identityBase + "/identity/users/" + chi.URLParam(r, "id")
	}))
	router.Put("/users/{id}", g.proxy(http.MethodPut, func(r *http.Request) string {
		return g.identityBase + "/identity/users/" + chi.URLParam(r, "id")
	}))
	router.Delete("/users/{id}", g.proxy(http.MethodDelete, func(r *http.Request) string {
		return g.identityBase + "/identity/users/" + chi.URLParam(r, "id")
	}))

	router.Get("/workflows", g.proxy(http.MethodGet, func(r *http.Request) string {
		return g.workflowBase + "/workflows"
	}))
	router.Post("/workflows", g.proxy(http.MethodPost, func(r *http.Request) string {
		return g.workflowBase + "/workflows"
	}))
	router.Get("/workflows/{id}", g.proxy(http.MethodGet, func(r *http.Request) string {
		return g.workflowBase + "/workflows/" + chi.URLParam(r, "id")
	}))
	router.Put("/workflows/{id}", g.proxy(http.MethodPut, func(r *http.Request) string {
		return g.workflowBase + "/workflows/" + chi.URLParam(r, "id")
	}))
	router.Delete("/workflows/{id}", g.proxy(http.MethodDelete, func(r *http.Request) string {
		return g.workflowBase + "/workflows/" + chi.URLParam(r, "id")
	}))
	router.Post("/workflows/{id}/publish", g.proxy(http.MethodPost, func(r *http.Request) string {
		return g.workflowBase + "/workflows/" + chi.URLParam(r, "id") + "/publish"
	}))

	router.Get("/overview", g.overviewHandler)
}

func (g *gateway) proxy(method string, target func(*http.Request) string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		g.forward(w, r, method, target(r))
	}
}

func (g *gateway) forward(w http.ResponseWriter, r *http.Request, method, target string) {
	var body io.Reader
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		rawBody, err := io.ReadAll(r.Body)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "failed to read request body")
			return
		}
		body = bytes.NewReader(rawBody)
	}

	req, err := http.NewRequestWithContext(r.Context(), method, target, body)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("failed to build upstream request: %v", err))
		return
	}

	copyRequestHeaders(req.Header, r.Header)
	appendForwardedHeaders(req.Header, r)
	req.URL.RawQuery = r.URL.RawQuery

	resp, err := g.client.Do(req)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, fmt.Sprintf("upstream request failed: %v", err))
		return
	}
	defer resp.Body.Close()

	for key := range w.Header() {
		w.Header().Del(key)
	}
	for key, values := range resp.Header {
		if strings.EqualFold(key, "Content-Length") {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	if resp.StatusCode == http.StatusNoContent {
		return
	}
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("gateway: failed to copy response body: %v", err)
	}
}

func (g *gateway) overviewHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	forms, err := g.fetchList(ctx, g.formBase+"/forms")
	if err != nil {
		g.renderUpstreamError(w, err)
		return
	}

	tickets, err := g.fetchList(ctx, g.ticketBase+"/tickets")
	if err != nil {
		g.renderUpstreamError(w, err)
		return
	}

	users, err := g.fetchList(ctx, g.identityBase+"/identity/users")
	if err != nil {
		g.renderUpstreamError(w, err)
		return
	}

	workflows, err := g.fetchList(ctx, g.workflowBase+"/workflows")
	if err != nil {
		g.renderUpstreamError(w, err)
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

	httpx.JSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"forms": map[string]any{
				"total": len(forms),
			},
			"tickets": map[string]any{
				"total":    len(tickets),
				"byStatus": ticketStatus,
			},
			"users": map[string]any{
				"total": len(users),
			},
			"workflows": map[string]any{
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

func (g *gateway) renderUpstreamError(w http.ResponseWriter, err error) {
	log.Printf("gateway: overview failed: %v", err)
	httpx.Error(w, http.StatusBadGateway, err.Error())
}

func trimTrailingSlash(value string) string {
	return strings.TrimRight(value, "/")
}

var hopHeaders = map[string]struct{}{
	"Connection":          {},
	"Proxy-Connection":    {},
	"Keep-Alive":          {},
	"Proxy-Authenticate":  {},
	"Proxy-Authorization": {},
	"Te":                  {},
	"Trailer":             {},
	"Transfer-Encoding":   {},
	"Upgrade":             {},
	"Content-Length":      {},
}

func copyRequestHeaders(dst, src http.Header) {
	for key, values := range src {
		canonical := http.CanonicalHeaderKey(key)
		if _, skip := hopHeaders[canonical]; skip {
			continue
		}
		for _, value := range values {
			dst.Add(canonical, value)
		}
	}
}

func appendForwardedHeaders(dst http.Header, r *http.Request) {
	if prior := r.Header.Get("X-Forwarded-For"); prior != "" {
		dst.Set("X-Forwarded-For", prior)
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}

	if host != "" {
		if prior := dst.Get("X-Forwarded-For"); prior != "" {
			dst.Set("X-Forwarded-For", prior+", "+host)
		} else {
			dst.Set("X-Forwarded-For", host)
		}
	}

	if r.Host != "" {
		dst.Set("X-Forwarded-Host", r.Host)
	}

	if proto := forwardedProto(r); proto != "" {
		dst.Set("X-Forwarded-Proto", proto)
	}
}

func forwardedProto(r *http.Request) string {
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto
	}
	if r.TLS != nil {
		return "https"
	}
	if r.URL != nil && r.URL.Scheme != "" {
		return r.URL.Scheme
	}
	return "http"
}

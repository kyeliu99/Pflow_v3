package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/pflow/gateway/internal/config"
	"github.com/pflow/gateway/internal/proxy"
)

// New constructs the HTTP server wiring for the gateway.
func New(cfg config.Config) *http.Server {
	client := &http.Client{Timeout: cfg.RequestTimeout}

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(cfg.RequestTimeout + time.Second))
	router.Use(middleware.StripSlashes)

	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	router.Route("/api", func(api chi.Router) {
		api.Get("/overview", overviewHandler(client, cfg))

		mountCollectionProxy(api, "/forms", ensureTrailingSlash(cfg.FormServiceURL+"/api/forms"), client)
		mountCollectionProxy(api, "/users", ensureTrailingSlash(cfg.IdentityServiceURL+"/api/users"), client)
		mountTicketRoutes(api, cfg, client)
		mountCollectionProxy(api, "/workflows", ensureTrailingSlash(cfg.WorkflowServiceURL+"/api/workflows"), client)
	})

	addr := fmt.Sprintf(":%s", cfg.Port)
	return &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.RequestTimeout + time.Second,
		WriteTimeout: cfg.RequestTimeout + time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func mountCollectionProxy(api chi.Router, prefix string, upstream string, client *http.Client) {
	api.MethodFunc(http.MethodGet, prefix, proxyHandler(prefix, upstream, client))
	api.MethodFunc(http.MethodPost, prefix, proxyHandler(prefix, upstream, client))
	api.MethodFunc(http.MethodGet, prefix+"/{id}", proxyHandler(prefix, upstream, client))
	api.MethodFunc(http.MethodPut, prefix+"/{id}", proxyHandler(prefix, upstream, client))
	api.MethodFunc(http.MethodPatch, prefix+"/{id}", proxyHandler(prefix, upstream, client))
	api.MethodFunc(http.MethodDelete, prefix+"/{id}", proxyHandler(prefix, upstream, client))
}

func mountTicketRoutes(api chi.Router, cfg config.Config, client *http.Client) {
	base := ensureTrailingSlash(cfg.TicketServiceURL + "/api/tickets")
	api.MethodFunc(http.MethodGet, "/tickets", proxyHandler("/tickets", base, client))
	api.MethodFunc(http.MethodPost, "/tickets", proxyHandler("/tickets", base, client))
	api.MethodFunc(http.MethodGet, "/tickets/{ticketID}", proxyHandler("/tickets", base, client))
	api.MethodFunc(http.MethodPut, "/tickets/{ticketID}", proxyHandler("/tickets", base, client))
	api.MethodFunc(http.MethodPatch, "/tickets/{ticketID}", proxyHandler("/tickets", base, client))
	api.MethodFunc(http.MethodDelete, "/tickets/{ticketID}", proxyHandler("/tickets", base, client))
	api.MethodFunc(http.MethodPost, "/tickets/{ticketID}/resolve", proxyHandler("/tickets", base, client))

	submissionsBase := ensureTrailingSlash(cfg.TicketServiceURL + "/api/tickets/submissions")
	api.MethodFunc(http.MethodGet, "/tickets/submissions", proxyHandler("/tickets/submissions", submissionsBase, client))
	api.MethodFunc(http.MethodPost, "/tickets/submissions", proxyHandler("/tickets/submissions", submissionsBase, client))
	api.MethodFunc(http.MethodGet, "/tickets/submissions/{submissionID}", proxyHandler("/tickets/submissions", submissionsBase, client))

	queueMetrics := ensureTrailingSlash(cfg.TicketServiceURL + "/api/tickets/queue-metrics")
	api.MethodFunc(http.MethodGet, "/tickets/queue-metrics", proxyHandler("/tickets/queue-metrics", queueMetrics, client))
}

func proxyHandler(prefix string, upstream string, client *http.Client) http.HandlerFunc {
	apiPrefix := "/api" + strings.TrimSuffix(prefix, "/")
	return func(w http.ResponseWriter, r *http.Request) {
		suffix := strings.TrimPrefix(r.URL.Path, apiPrefix)
		if suffix == "" || suffix == "/" {
			suffix = ""
		}
		if suffix != "" && !strings.HasPrefix(suffix, "/") {
			suffix = "/" + suffix
		}
		proxy.Forward(w, r, client, upstream, suffix)
	}
}

func overviewHandler(client *http.Client, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		forms := fetchCollection(client, ensureTrailingSlash(cfg.FormServiceURL+"/api/forms"))
		tickets := fetchCollection(client, ensureTrailingSlash(cfg.TicketServiceURL+"/api/tickets"))
		users := fetchCollection(client, ensureTrailingSlash(cfg.IdentityServiceURL+"/api/users"))
		workflows := fetchCollection(client, ensureTrailingSlash(cfg.WorkflowServiceURL+"/api/workflows"))

		queueMetrics := map[string]any{
			"pending":              0,
			"processing":           0,
			"completed":            0,
			"failed":               0,
			"oldestPendingSeconds": 0,
		}

		if metrics, err := fetchObject(client, ensureTrailingSlash(cfg.TicketServiceURL+"/api/tickets/queue-metrics")); err == nil {
			queueMetrics = metrics
		}

		ticketStatusCounts := map[string]int{}
		for _, ticket := range tickets {
			if status, ok := ticket["status"].(string); ok {
				ticketStatusCounts[status]++
			}
		}

		published := 0
		for _, workflow := range workflows {
			if active, ok := workflow["is_active"].(bool); ok && active {
				published++
			}
		}

		payload := map[string]any{
			"forms": map[string]any{"total": len(forms)},
			"tickets": map[string]any{
				"total":    len(tickets),
				"byStatus": ticketStatusCounts,
				"queue":    queueMetrics,
			},
			"users": map[string]any{"total": len(users)},
			"workflows": map[string]any{
				"total":     len(workflows),
				"published": published,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}
}

func fetchCollection(client *http.Client, url string) []map[string]any {
	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	if err != nil {
		return []map[string]any{}
	}

	resp, err := client.Do(req)
	if err != nil {
		return []map[string]any{}
	}
	body, err := proxy.ReadBody(resp)
	if err != nil {
		return []map[string]any{}
	}

	return proxy.DecodeJSONArray(body)
}

func fetchObject(client *http.Client, url string) (map[string]any, error) {
	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := proxy.ReadBody(resp)
	if err != nil {
		return nil, err
	}
	obj, err := proxy.DecodeObject(body)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func ensureTrailingSlash(value string) string {
	if strings.HasSuffix(value, "/") {
		return value
	}
	return value + "/"
}

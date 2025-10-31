package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config captures all runtime configuration knobs for the gateway.
type Config struct {
	Port                string
	FormServiceURL      string
	IdentityServiceURL  string
	TicketServiceURL    string
	WorkflowServiceURL  string
	RequestTimeout      time.Duration
	ShutdownGracePeriod time.Duration
}

const (
	defaultPort           = "8000"
	defaultRequestTimeout = 5 * time.Second
	defaultShutdownGrace  = 5 * time.Second
)

// Load parses configuration from environment variables and applies defaults when
// values are omitted. It performs basic validation to ensure required upstreams
// are provided.
func Load() (Config, error) {
	cfg := Config{
		Port:                getEnv("GATEWAY_PORT", defaultPort),
		FormServiceURL:      os.Getenv("FORM_SERVICE_URL"),
		IdentityServiceURL:  os.Getenv("IDENTITY_SERVICE_URL"),
		TicketServiceURL:    os.Getenv("TICKET_SERVICE_URL"),
		WorkflowServiceURL:  os.Getenv("WORKFLOW_SERVICE_URL"),
		RequestTimeout:      parseDuration("GATEWAY_REQUEST_TIMEOUT", defaultRequestTimeout),
		ShutdownGracePeriod: parseDuration("GATEWAY_SHUTDOWN_GRACE", defaultShutdownGrace),
	}

	if cfg.FormServiceURL == "" {
		return Config{}, fmt.Errorf("FORM_SERVICE_URL is required")
	}
	if cfg.IdentityServiceURL == "" {
		return Config{}, fmt.Errorf("IDENTITY_SERVICE_URL is required")
	}
	if cfg.TicketServiceURL == "" {
		return Config{}, fmt.Errorf("TICKET_SERVICE_URL is required")
	}
	if cfg.WorkflowServiceURL == "" {
		return Config{}, fmt.Errorf("WORKFLOW_SERVICE_URL is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func parseDuration(key string, fallback time.Duration) time.Duration {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback
	}

	// Allow values in seconds for convenience.
	if seconds, err := strconv.Atoi(raw); err == nil {
		return time.Duration(seconds) * time.Second
	}

	if d, err := time.ParseDuration(raw); err == nil {
		return d
	}

	return fallback
}

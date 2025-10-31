package config

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/joho/godotenv"
)

const defaultHTTPPort = "8080"

// AppConfig captures environment variables shared across services.
type AppConfig struct {
	ServiceName        string
	HTTPPort           string
	PostgresDSN        string
	KafkaBrokers       string
	KafkaTopic         string
	CamundaURL         string
	FormServiceURL     string
	IdentityServiceURL string
	TicketServiceURL   string
	WorkflowServiceURL string

	ServiceDatabaseDSN  map[string]string
	ServiceHTTPPorts    map[string]string
	ServiceKafkaBrokers map[string]string
	ServiceQueueTopics  map[string]string
	ServiceQueueGroups  map[string]string
}

var (
	once sync.Once
	cfg  *AppConfig
)

// Load reads environment variables, optionally from a .env file.
func Load() *AppConfig {
	once.Do(func() {
		loadEnvFiles()

		cfg = &AppConfig{
			ServiceName:        getEnv("SERVICE_NAME", defaultServiceName()),
			HTTPPort:           getEnv("HTTP_PORT", defaultHTTPPort),
			PostgresDSN:        getEnv("POSTGRES_DSN", "postgres://pflow:pflow@localhost:5432/pflow?sslmode=disable"),
			KafkaBrokers:       getEnv("KAFKA_BROKERS", "localhost:9092"),
			KafkaTopic:         getEnv("KAFKA_TOPIC", "pflow-events"),
			CamundaURL:         getEnv("CAMUNDA_URL", "localhost:26500"),
			FormServiceURL:     getEnv("FORM_SERVICE_URL", "http://localhost:8081"),
			IdentityServiceURL: getEnv("IDENTITY_SERVICE_URL", "http://localhost:8082"),
			TicketServiceURL:   getEnv("TICKET_SERVICE_URL", "http://localhost:8083"),
			WorkflowServiceURL: getEnv("WORKFLOW_SERVICE_URL", "http://localhost:8084"),
		}

		cfg.ServiceDatabaseDSN = collectServiceValues("DATABASE_DSN")
		cfg.ServiceHTTPPorts = collectServiceValues("HTTP_PORT")
		cfg.ServiceKafkaBrokers = collectServiceValues("KAFKA_BROKERS")
		cfg.ServiceQueueTopics = collectServiceValues("QUEUE_TOPIC")
		cfg.ServiceQueueGroups = collectServiceValues("QUEUE_GROUP")
	})

	return cfg
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func defaultServiceName() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Base(exe)
	}
	return "pflow-service"
}

func loadEnvFiles() {
	files := uniqueStrings(expandEnvFiles())
	for _, file := range files {
		if file == "" {
			continue
		}
		if _, err := os.Stat(file); err != nil {
			continue
		}
		if err := godotenv.Overload(file); err != nil {
			log.Printf("config: failed to load %s: %v", file, err)
		}
	}
}

func expandEnvFiles() []string {
	files := []string{".env"}

	if extra := os.Getenv("PFLOW_ENV_FILES"); extra != "" {
		files = append(files, strings.Split(extra, ",")...)
	}

	if repoRoot := locateRepoRoot(); repoRoot != "" {
		files = append(files,
			filepath.Join(repoRoot, ".env"),
			filepath.Join(repoRoot, ".env.local"),
		)

		envDir := filepath.Join(repoRoot, ".env.d")
		entries, err := os.ReadDir(envDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				if strings.HasSuffix(entry.Name(), ".env") {
					files = append(files, filepath.Join(envDir, entry.Name()))
				}
			}
		}
	}

	return files
}

func locateRepoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		if dir == "" || dir == "/" {
			return ""
		}

		if fileExists(filepath.Join(dir, "go.work")) || fileExists(filepath.Join(dir, ".git")) {
			return dir
		}

		dir = filepath.Dir(dir)
	}
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// IsEnvSet reports whether an environment variable was explicitly provided.
func IsEnvSet(key string) bool {
	_, ok := os.LookupEnv(key)
	return ok
}

// ResolveHTTPPort returns the configured HTTP port or a service-specific default.
func (cfg *AppConfig) ResolveHTTPPort(fallback string) string {
	if cfg == nil {
		if fallback == "" {
			return defaultHTTPPort
		}
		return fallback
	}

	port := strings.TrimSpace(cfg.HTTPPort)
	if port == "" {
		if fallback == "" {
			return defaultHTTPPort
		}
		return fallback
	}

	if port == defaultHTTPPort && !IsEnvSet("HTTP_PORT") {
		if fallback == "" {
			return defaultHTTPPort
		}
		return fallback
	}

	return port
}

// ResolveServiceHTTPPort resolves a service-scoped HTTP port with fallback support.
func (cfg *AppConfig) ResolveServiceHTTPPort(service, fallback string) string {
	if cfg == nil {
		if fallback == "" {
			return defaultHTTPPort
		}
		return fallback
	}

	serviceKey := normalizeServiceKey(service)
	if port, ok := cfg.ServiceHTTPPorts[serviceKey]; ok {
		port = strings.TrimSpace(port)
		if port != "" {
			return port
		}
	}

	return cfg.ResolveHTTPPort(fallback)
}

// KafkaBrokerList resolves a service-specific Kafka broker list, falling back to the shared setting.
func (cfg *AppConfig) KafkaBrokerList(service string) []string {
	var raw string
	if cfg != nil {
		raw = cfg.KafkaBrokers
		serviceKey := normalizeServiceKey(service)
		if brokers, ok := cfg.ServiceKafkaBrokers[serviceKey]; ok {
			brokers = strings.TrimSpace(brokers)
			if brokers != "" {
				raw = brokers
			}
		}
	}

	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// ResolveServiceQueueTopic returns the topic name for the service-specific queue.
func (cfg *AppConfig) ResolveServiceQueueTopic(service, fallback string) string {
	if cfg == nil {
		return strings.TrimSpace(fallback)
	}

	serviceKey := normalizeServiceKey(service)
	if topic, ok := cfg.ServiceQueueTopics[serviceKey]; ok {
		topic = strings.TrimSpace(topic)
		if topic != "" {
			return topic
		}
	}

	topic := strings.TrimSpace(fallback)
	if topic != "" {
		return topic
	}
	return strings.TrimSpace(cfg.KafkaTopic)
}

// ResolveServiceQueueGroup returns the consumer group for the service queue.
func (cfg *AppConfig) ResolveServiceQueueGroup(service, fallback string) string {
	if cfg == nil {
		return strings.TrimSpace(fallback)
	}

	serviceKey := normalizeServiceKey(service)
	if group, ok := cfg.ServiceQueueGroups[serviceKey]; ok {
		group = strings.TrimSpace(group)
		if group != "" {
			return group
		}
	}

	return strings.TrimSpace(fallback)
}

// DatabaseDSN resolves the database DSN for a service, defaulting to PostgresDSN.
func (cfg *AppConfig) DatabaseDSN(service string) string {
	if cfg == nil {
		return ""
	}

	serviceKey := normalizeServiceKey(service)
	if dsn, ok := cfg.ServiceDatabaseDSN[serviceKey]; ok {
		dsn = strings.TrimSpace(dsn)
		if dsn != "" {
			return dsn
		}
	}

	return cfg.PostgresDSN
}

// MustGet returns the loaded configuration or exits the process.
func MustGet() *AppConfig {
	if cfg == nil {
		log.Fatal("config not loaded")
	}
	return cfg
}

func collectServiceValues(suffix string) map[string]string {
	values := make(map[string]string)
	normalizedSuffix := "_" + strings.ToUpper(strings.TrimSpace(suffix))

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		key := parts[0]
		val := ""
		if len(parts) == 2 {
			val = parts[1]
		}

		upperKey := strings.ToUpper(strings.TrimSpace(key))
		if !strings.HasSuffix(upperKey, normalizedSuffix) {
			continue
		}

		name := strings.TrimSuffix(upperKey, normalizedSuffix)
		name = strings.TrimPrefix(name, "PFLOW_")
		name = strings.Trim(name, "_")
		if name == "" {
			continue
		}

		values[strings.ToLower(name)] = strings.TrimSpace(val)
	}

	return values
}

func normalizeServiceKey(service string) string {
	key := strings.ToLower(strings.TrimSpace(service))
	key = strings.ReplaceAll(key, "-", "_")
	return key
}

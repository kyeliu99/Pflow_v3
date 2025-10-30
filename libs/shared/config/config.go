package config

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/joho/godotenv"
)

// AppConfig captures environment variables shared across services.
type AppConfig struct {
	ServiceName  string
	HTTPPort     string
	PostgresDSN  string
	KafkaBrokers string
	KafkaTopic   string
	CamundaURL   string
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
			ServiceName:  getEnv("SERVICE_NAME", defaultServiceName()),
			HTTPPort:     getEnv("HTTP_PORT", "8080"),
			PostgresDSN:  getEnv("POSTGRES_DSN", "postgres://pflow:pflow@localhost:5432/pflow?sslmode=disable"),
			KafkaBrokers: getEnv("KAFKA_BROKERS", "localhost:9092"),
			KafkaTopic:   getEnv("KAFKA_TOPIC", "pflow-events"),
			CamundaURL:   getEnv("CAMUNDA_URL", "localhost:26500"),
		}
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

// MustGet returns the loaded configuration or exits the process.
func MustGet() *AppConfig {
	if cfg == nil {
		log.Fatal("config not loaded")
	}
	return cfg
}

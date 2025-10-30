package config

import (
	"log"
	"os"
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
		_ = godotenv.Load()

		cfg = &AppConfig{
			ServiceName:  getEnv("SERVICE_NAME", "pflow-service"),
			HTTPPort:     getEnv("HTTP_PORT", "8080"),
			PostgresDSN:  getEnv("POSTGRES_DSN", "postgres://pflow:pflow@localhost:5432/pflow?sslmode=disable"),
			KafkaBrokers: getEnv("KAFKA_BROKERS", "localhost:9092"),
			KafkaTopic:   getEnv("KAFKA_TOPIC", "pflow-events"),
			CamundaURL:   getEnv("CAMUNDA_URL", "http://localhost:8088"),
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

// MustGet returns the loaded configuration or exits the process.
func MustGet() *AppConfig {
	if cfg == nil {
		log.Fatal("config not loaded")
	}
	return cfg
}

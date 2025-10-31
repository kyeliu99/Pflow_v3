package database

import (
	"log"
	"sync"

	"github.com/pflow/shared/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	mu          sync.Mutex
	connections = make(map[string]*gorm.DB)
	defaultDB   *gorm.DB
)

// Connect initializes a singleton PostgreSQL connection using GORM.
func Connect() *gorm.DB {
	cfg := config.MustGet()
	return ConnectWithDSN("default", cfg.PostgresDSN)
}

// ConnectWithDSN initialises or returns a named PostgreSQL connection.
func ConnectWithDSN(name, dsn string) *gorm.DB {
	key := name
	if key == "" {
		key = dsn
	}

	mu.Lock()
	defer mu.Unlock()

	if db, ok := connections[key]; ok {
		return db
	}

	if dsn == "" {
		log.Fatalf("database: DSN not provided for connection %s", key)
	}

	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to postgres (%s): %v", key, err)
	}

	connections[key] = conn
	if name == "default" || defaultDB == nil {
		defaultDB = conn
	}

	return conn
}

// DB returns the initialized default database or nil if Connect was not called.
func DB() *gorm.DB {
	mu.Lock()
	defer mu.Unlock()
	return defaultDB
}

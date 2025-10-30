package database

import (
	"log"
	"sync"

	"github.com/pflow/shared/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	once sync.Once
	db   *gorm.DB
)

// Connect initializes a singleton PostgreSQL connection using GORM.
func Connect() *gorm.DB {
	once.Do(func() {
		cfg := config.MustGet()
		dsn := cfg.PostgresDSN
		conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Fatalf("failed to connect to postgres: %v", err)
		}
		db = conn
	})

	return db
}

// DB returns the initialized database or nil if Connect was not called.
func DB() *gorm.DB {
	return db
}

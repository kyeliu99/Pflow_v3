module github.com/pflow/components

go 1.21

require (
	github.com/go-chi/chi/v5 v5.1.0
	github.com/google/uuid v1.6.0
	github.com/pflow/shared v0.0.0
	gorm.io/datatypes v1.2.7
	gorm.io/gorm v1.30.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/go-sql-driver/mysql v1.8.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/text v0.20.0 // indirect
	gorm.io/driver/mysql v1.5.6 // indirect
)

replace github.com/pflow/shared => ../shared

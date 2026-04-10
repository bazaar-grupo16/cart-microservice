package config

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

var DB *sql.DB

// ConnectDatabase establishes a connection to the PostgreSQL database using environment variables and assigns it to the global DB variable. It also logs the connection status.
func ConnectDatabase() {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DATABASE_HOST"),
		os.Getenv("DATABASE_PORT"),
		os.Getenv("DATABASE_USER"),
		os.Getenv("DATABASE_PASSWORD"),
		os.Getenv("DATABASE_NAME"),
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		zap.S().Fatal("Error opening DB:", err)
	}

	if err = db.Ping(); err != nil {
		zap.S().Fatal("Error connecting to DB:", err)
	}

	DB = db
	zap.S().Info("Connected to PostgreSQL")
}

// RunMigrations runs the database migrations using the golang-migrate library. It constructs the DSN for the PostgreSQL database, initializes the migrator with the file source for migrations, and applies any pending migrations. It also logs the migration status.
func RunMigrations() {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("DATABASE_USER"),
		os.Getenv("DATABASE_PASSWORD"),
		os.Getenv("DATABASE_HOST"),
		os.Getenv("DATABASE_PORT"),
		os.Getenv("DATABASE_NAME"),
	)

	wd, _ := os.Getwd()
	m, err := migrate.New("file://"+wd+"/db/migrations", dsn)
	if err != nil {
		zap.S().Fatal("Error creating migrator:", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		zap.S().Fatal("Error running migrations:", err)
	}

	zap.S().Info("Migrations applied successfully")
}

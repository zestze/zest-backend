package zql

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	pgx5 "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func Sqlite3(dbName string) (*sql.DB, error) {
	return sql.Open("sqlite3", dbName)
}

func Postgres() (*sql.DB, error) {
	return sql.Open("pgx", defaultConfig().String())
}

func WithMigrations() (*sql.DB, error) {
	db, err := Postgres()
	if err != nil {
		return nil, err
	}

	driver, err := pgx5.WithInstance(db, &pgx5.Config{})
	if err != nil {
		return nil, err
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"pgx5", driver)
	if err != nil {
		return nil, err
	}

	slog.Info("running migrations")
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, err
	}
	slog.Info("migrations successful")
	return db, nil
}

type postgresConfig struct {
	host     string
	port     int
	dbname   string
	username string
	password string
}

// String returns config as DSN
func (cfg postgresConfig) String() string {
	return fmt.Sprintf(
		`user=%v password=%v host=%v port=%v database=%v sslmode=disable`,
		cfg.username, cfg.password, cfg.host, cfg.port, cfg.dbname)
}

func defaultConfig() postgresConfig {
	return postgresConfig{
		host:     "postgres",
		port:     5432,
		dbname:   "zest",
		username: "zeke",
		password: "reyna",
	}
}

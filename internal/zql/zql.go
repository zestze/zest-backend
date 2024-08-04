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
	return PostgresWithConfig(defaultConfig())
}

// TODO(zeke): deprecate and remove!
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

func PostgresWithOptions(opts ...func(cfg *PostgresConfig)) (*sql.DB, error) {
	cfg := defaultConfig()

	for _, o := range opts {
		o(&cfg)
	}

	return sql.Open("pgx", cfg.String())
}

func WithHost(host string) func(cfg *PostgresConfig) {
	return func(cfg *PostgresConfig) {
		cfg.host = host
	}
}

func WithDatabase(dbname string) func(cfg *PostgresConfig) {
	return func(cfg *PostgresConfig) {
		cfg.dbname = dbname
	}
}

func PostgresWithConfig(cfg PostgresConfig) (*sql.DB, error) {
	return sql.Open("pgx", cfg.String())
}

type PostgresConfig struct {
	host     string
	port     int
	dbname   string
	username string
	password string
}

// String returns config as DSN
func (cfg PostgresConfig) String() string {
	return fmt.Sprintf(
		`user=%v password=%v host=%v port=%v database=%v sslmode=disable`,
		cfg.username, cfg.password, cfg.host, cfg.port, cfg.dbname)
}

func defaultConfig() PostgresConfig {
	return PostgresConfig{
		host:     "postgres",
		port:     5432,
		dbname:   "zest",
		username: "zeke",
		password: "reyna",
	}
}

// Rollback just rolls back the current transaction and joins the error to the original err if applicable
func Rollback(tx *sql.Tx, originalErr error) error {
	return errors.Join(originalErr, tx.Rollback())
}

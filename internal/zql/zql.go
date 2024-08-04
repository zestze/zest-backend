package zql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"

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

// ForTesting prepares an environment for unit tests.
//
// dbName is the name of the database to create for this test.
// schemaPath is the path relative to the testing working directory to the schema.sql file to use for migrating
func ForTesting(
	ctx context.Context, dbName, hostname, schemaPath string, dropDB bool,
) (db *sql.DB, toDefer func(), err error) {
	// TODO(zeke): pass `t` to func and mark as helper?
	parentDB, err := PostgresWithOptions(WithHost(hostname))
	if err != nil {
		return
	}
	defer parentDB.Close()

	_, err = parentDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %v;", dbName))
	if err != nil {
		return
	}

	db, err = PostgresWithOptions(WithHost(hostname), WithDatabase(dbName))
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			db.Close()
		}
	}()

	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		return
	}

	_, err = db.ExecContext(ctx, string(schema))
	if err != nil {
		return
	}

	toDefer = func() {
		if !dropDB {
			return
		}

		// postgres docs say:
		// 	You cannot be connected to the database you are about to remove.
		//	Instead, connect to template1 or any other database and run this command again.
		tempDB, err := PostgresWithOptions(WithHost(hostname), WithDatabase("template1"))
		if err != nil {
			slog.Error("error dropping test db, when opening new connection: %v")
		}
		defer tempDB.Close()
		_, err = tempDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %v;", dbName))
		if err != nil {
			slog.Error("error dropping test db", "error", err)
		}
	}
	return
}

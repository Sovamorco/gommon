package gsqlx

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // need as sqlx driver.
	"github.com/sovamorco/errorx"
	_ "modernc.org/sqlite" // need as sqlx driver.
)

type Driver string

const (
	Postgres Driver = "pg"
	Memory   Driver = "memory"
)

type Config struct {
	Driver Driver `mapstructure:"driver"`

	Postgres PostgresConfig `mapstructure:"pg"`
}

type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	DB       string `mapstructure:"db"`
	SSL      bool   `mapstructure:"ssl"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

func New(ctx context.Context, cfg Config) (*sqlx.DB, error) {
	var db *sqlx.DB

	var err error

	switch cfg.Driver {
	case Postgres:
		db, err = pgInit(ctx, cfg.Postgres)
	case Memory:
		db, err = memoryInit(ctx)
	default:
		err = errorx.IllegalArgument.New("unknown database driver: %s", cfg.Driver)
	}

	if err != nil {
		return nil, errorx.Wrap(err, "create database from driver")
	}

	return db, nil
}

func pgInit(ctx context.Context, cfg PostgresConfig) (*sqlx.DB, error) {
	db, err := sqlx.ConnectContext(ctx, "postgres", makeDSN(cfg))
	if err != nil {
		return nil, errorx.Wrap(err, "connect to postgres db")
	}

	return db, nil
}

func makeDSN(cfg PostgresConfig) string {
	sslmode := "disable"
	if cfg.SSL {
		sslmode = "require"
	}

	return fmt.Sprintf(
		"host=%s port=%d dbname=%s sslmode=%s user=%s password=%s",
		cfg.Host, cfg.Port, cfg.DB, sslmode, cfg.Username, cfg.Password,
	)
}

func memoryInit(ctx context.Context) (*sqlx.DB, error) {
	db, err := sqlx.ConnectContext(ctx, "sqlite", ":memory:")
	if err != nil {
		return nil, errorx.Wrap(err, "connect to sqlite in-memory db")
	}

	// sqlite in-memory does not support concurrent connections.
	db.SetMaxOpenConns(1)

	return db, nil
}

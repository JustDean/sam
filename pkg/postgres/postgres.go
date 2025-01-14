package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	DbName   string
}

func (c *Config) url() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s", c.Username, c.Password, c.Host, c.Port, c.DbName)
}

func SetPostgresPool(c Config) (*pgxpool.Pool, error) {
	dbpool, err := pgxpool.New(context.Background(), c.url())
	if err != nil {
		return nil, err
	}
	if err := dbpool.Ping(context.Background()); err != nil {
		return nil, err
	}
	return dbpool, err
}

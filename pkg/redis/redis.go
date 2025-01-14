package redis

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	Host     string
	Port     string
	Password string
	Db       string // should be a positive number (0-11)
}

func (rc *Config) addr() string {
	return fmt.Sprintf("%s:%s", rc.Host, rc.Port)
}

func SetRedisPool(config Config) (*redis.Client, error) {
	dbNumber, err := strconv.ParseInt(config.Db, 10, 32)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(&redis.Options{
		Addr:     config.addr(),
		Password: config.Password,
		DB:       int(dbNumber),
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return client, nil
}

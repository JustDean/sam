package grpc

import "fmt"

type Config struct {
	Host string
	Port string
}

func (c *Config) url() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

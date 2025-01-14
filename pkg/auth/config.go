package auth

import (
	"github.com/JustDean/sam/pkg/postgres"
	redis_utils "github.com/JustDean/sam/pkg/redis"
)

type AuthManagerConfig struct {
	Db    postgres.Config
	Cache redis_utils.Config
}

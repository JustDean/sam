package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/JustDean/sam/grpc"
	"github.com/JustDean/sam/pkg/auth"
	"github.com/JustDean/sam/pkg/postgres"
	"github.com/JustDean/sam/pkg/redis"
	"github.com/JustDean/sam/pkg/utils"
)

func main() {
	log.Println("Starting the app")
	authManager, err := auth.SetAuthManager(auth.AuthManagerConfig{
		Db: postgres.Config{
			Host:     utils.GetEnv("DB_HOST", "localhost"),
			Port:     utils.GetEnv("DB_PORT", "5432"),
			Username: utils.GetEnv("DB_USERNAME", "sam"),
			Password: utils.GetEnv("DB_PASSWORD", "sam"),
			DbName:   utils.GetEnv("DB_NAME", "sam"),
		},
		Cache: redis.Config{
			Host:     utils.GetEnv("CACHE_HOST", "localhost"),
			Port:     utils.GetEnv("CACHE_PORT", "6379"),
			Password: utils.GetEnv("CACHE_PASSWORD", ""),
			Db:       utils.GetEnv("CACHE_DB", "1"),
		},
	})
	if err != nil {
		log.Fatalf("Error setting Auth Manager %v", err)
	}
	server, err := grpc.SetServer(grpc.Config{
		Host: utils.GetEnv("SERVER_HOST", "localhost"),
		Port: utils.GetEnv("SERVER_PORT", "9999"),
	}, authManager)
	if err != nil {
		log.Fatalf("Error setting gRPC server %v", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		authManager.Run(ctx)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		server.Run(ctx)
	}()
	wg.Wait()
	log.Println("Service is shut down.")
}

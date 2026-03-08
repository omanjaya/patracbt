package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/omanjaya/patra/config"
	"github.com/omanjaya/patra/pkg/logger"
	"github.com/redis/go-redis/v9"
)

func NewClient(cfg *config.Config) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,

		// Connection pool tuning.
		PoolSize:        50,
		MinIdleConns:    10,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		ConnMaxLifetime: 30 * time.Minute,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		logger.Log.Fatalf("Gagal koneksi ke Redis: %v", err)
	}

	logger.Log.Info("Redis connected")
	return client
}

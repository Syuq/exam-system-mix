package utils

import (
	"context"
	"encoding/json"
	"exam-system/config"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

func InitRedis() (*RedisClient, error) {
	cfg := config.AppConfig.Redis

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx := context.Background()

	// Test connection
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{
		client: rdb,
		ctx:    ctx,
	}, nil
}

func (r *RedisClient) Set(key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(r.ctx, key, value, expiration).Err()
}

func (r *RedisClient) Get(key string) (string, error) {
	return r.client.Get(r.ctx, key).Result()
}

func (r *RedisClient) Del(key string) error {
	return r.client.Del(r.ctx, key).Err()
}

func (r *RedisClient) Exists(key string) (bool, error) {
	count, err := r.client.Exists(r.ctx, key).Result()
	return count > 0, err
}

func (r *RedisClient) SetJSON(key string, value interface{}, expiration time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return r.Set(key, jsonData, expiration)
}

func (r *RedisClient) GetJSON(key string, dest interface{}) error {
	jsonData, err := r.Get(key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(jsonData), dest)
}

func (r *RedisClient) Incr(key string) (int64, error) {
	return r.client.Incr(r.ctx, key).Result()
}

func (r *RedisClient) IncrBy(key string, value int64) (int64, error) {
	return r.client.IncrBy(r.ctx, key, value).Result()
}

func (r *RedisClient) Expire(key string, expiration time.Duration) error {
	return r.client.Expire(r.ctx, key, expiration).Err()
}

func (r *RedisClient) TTL(key string) (time.Duration, error) {
	return r.client.TTL(r.ctx, key).Result()
}

// Rate limiting using token bucket algorithm
func (r *RedisClient) IsRateLimited(key string, limit int, window time.Duration) (bool, error) {
	// Use sliding window log approach
	now := time.Now().Unix()
	windowStart := now - int64(window.Seconds())

	pipe := r.client.Pipeline()

	// Remove expired entries
	pipe.ZRemRangeByScore(r.ctx, key, "0", fmt.Sprintf("%d", windowStart))

	// Count current requests
	pipe.ZCard(r.ctx, key)

	// Add current request
	pipe.ZAdd(r.ctx, key, redis.Z{Score: float64(now), Member: now})

	// Set expiration
	pipe.Expire(r.ctx, key, window)

	results, err := pipe.Exec(r.ctx)
	if err != nil {
		return false, err
	}

	// Get count from the second command (ZCard)
	count := results[1].(*redis.IntCmd).Val()

	return count >= int64(limit), nil
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}


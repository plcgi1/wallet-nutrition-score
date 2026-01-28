package cache

import (
	"context"
	"encoding/json"
	"time"

	"wallet-nutrition-score/config"
	"wallet-nutrition-score/internal/entity"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

const (
	// Cache expiration time
	CacheExpiration = 5 * time.Minute
)

// Cache - Интерфейс для кэша
type Cache interface {
	GetWalletReport(ctx context.Context, address string) (*entity.WalletReport, error)
	SetWalletReport(ctx context.Context, address string, report *entity.WalletReport) error
	Close() error
}

// RedisCache - Реализация кэша с помощью Redis
type RedisCache struct {
	client *redis.Client
	log    *logrus.Entry
}

// NewRedisCache - Создает новый Redis кэш
func NewRedisCache(cfg *config.Config, log *logrus.Entry) (*RedisCache, error) {
	logger := log.WithFields(logrus.Fields{"component": "redis"})
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Проверка соединения с Redis
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		log.Errorf("Failed to connect to Redis: %v", err)
		return nil, err
	}

	logger.Info("Successfully connected to Redis")

	return &RedisCache{
		client: client,
		log:    logger,
	}, nil
}

// GetWalletReport - Получает отчет о кошельке из кэша
func (c *RedisCache) GetWalletReport(ctx context.Context, address string) (*entity.WalletReport, error) {
	key := c.getCacheKey(address)

	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		c.log.Debugf("Cache miss for address: %s", address)
		return nil, nil
	} else if err != nil {
		c.log.Errorf("Failed to get from cache: %v", err)
		return nil, err
	}

	var report entity.WalletReport
	err = json.Unmarshal([]byte(val), &report)
	if err != nil {
		c.log.Errorf("Failed to unmarshal cache value: %v", err)
		return nil, err
	}

	c.log.Debugf("Cache hit for address: %s", address)
	return &report, nil
}

// SetWalletReport - Сохраняет отчет о кошельке в кэш
func (c *RedisCache) SetWalletReport(ctx context.Context, address string, report *entity.WalletReport) error {
	key := c.getCacheKey(address)

	data, err := json.Marshal(report)
	if err != nil {
		c.log.Errorf("Failed to marshal report: %v", err)
		return err
	}

	err = c.client.Set(ctx, key, string(data), CacheExpiration).Err()
	if err != nil {
		c.log.Errorf("Failed to set cache: %v", err)
		return err
	}

	c.log.Debugf("Cache set for address: %s", address)
	return nil
}

// Close - Закрывает соединение с Redis
func (c *RedisCache) Close() error {
	if err := c.client.Close(); err != nil {
		c.log.Errorf("Failed to close Redis connection: %v", err)
		return err
	}
	c.log.Info("Redis connection closed")
	return nil
}

// getCacheKey - Генерирует ключ для кэша на основе адреса кошелька
func (c *RedisCache) getCacheKey(address string) string {
	return "wallet_report:" + address
}

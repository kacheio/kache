// MIT License
//
// Copyright (c) 2023 kache.io
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package provider

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

var (
	ErrRedisConfigNoEndpoint    = errors.New("no redis endpoint configured")
	ErrRedisMaxQueueConcurrency = errors.New("max job queue concurrency must be positive")
	ErrRedisMaxItemSize         = errors.New("max item size exceeded")
	ErrRedisJobQueueFull        = errors.New("job queue is full")
)

// RedisClientConfig holds the configuration for the Redis client.
type RedisClientConfig struct {
	// Endpoint holds the endpoint addresses of the Redis server.
	// Either a single address or a list of host:port addresses
	// of cluster/sentinel nodes.
	Endpoint string `yaml:"endpoint"`

	// Username to authenticate the current connection with one of the connections
	// defined in the ACL list when connecting to a Redis 6.0 instance, or greater,
	// that is using the Redis ACL system.
	Username string `yaml:"username"`

	// Optional password. Must match the password specified in the
	// requirepass server configuration option (if connecting to a Redis 5.0 instance, or lower),
	// or the User Password when connecting to a Redis 6.0 instance, or greater,
	// that is using the Redis ACL system.
	Password string `yaml:"password"`

	// DB Database to be selected after connecting to the server.
	DB int `yaml:"db"`

	// MaxItemSize specifies the maximum size of an item stored in Redis.
	// Items bigger than MaxItemSize are skipped. If set to 0, no maximum size is enforced.
	MaxItemSize int `yaml:"max_item_size"`

	// MaxQueueBufferSize is the maximum number of enqueued job operations allowed.
	MaxQueueBufferSize int `yaml:"max_queue_buffer_size"`

	// MaxQueueConcurrency is the maximum number of concurrent async job operations.
	MaxQueueConcurrency int `yaml:"max_queue_concurrency"`
}

// Validate validates the RedisClientConfig.
func (c *RedisClientConfig) Validate() error {
	if len(c.Endpoint) == 0 {
		return ErrRedisConfigNoEndpoint
	}
	if c.MaxQueueConcurrency < 1 {
		return ErrRedisMaxQueueConcurrency
	}
	return nil
}

// redisClient wraps a Redis Universal Client.
type redisClient struct {
	redis.UniversalClient

	// config is the configuration of the client.
	config RedisClientConfig

	// queue is the async job queue.
	queue *jobQueue
}

// NewRedisClient creates a new Redis client with the provided configuration.
func NewRedisClient(name string, config RedisClientConfig) (RemoteCacheClient, error) {
	opts := &redis.UniversalOptions{
		Addrs:    strings.Split(config.Endpoint, ","),
		Username: config.Username,
		Password: config.Password,
		DB:       config.DB,
	}
	c := &redisClient{
		UniversalClient: redis.NewUniversalClient(opts),
		config:          config,
		queue:           newJobQueue(config.MaxQueueBufferSize, config.MaxQueueConcurrency),
	}
	if err := c.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return c, nil
}

// Fetch performs a Redis Get operation.
func (c *redisClient) Fetch(ctx context.Context, key string) []byte {
	res, err := c.Get(ctx, key).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			log.Error().Err(err).Str("cache-key", key).Msg("Error getting item from redis")
		}
		return nil
	}
	return []byte(res)
}

// Store stores a key and value into Redis.
func (c *redisClient) Store(key string, value []byte, ttl time.Duration) error {
	if c.config.MaxItemSize > 0 && len(value) > c.config.MaxItemSize {
		return ErrRedisMaxItemSize
	}
	_, err := c.Set(context.Background(), key, value, ttl).Result()
	return err

}

// StoreAsync store a key and value into Redis asynchronously.
func (c *redisClient) StoreAsync(key string, value []byte, ttl time.Duration) error {
	err := c.queue.dispatch(func() {
		err := c.Store(key, value, ttl)
		if err != nil {
			log.Error().Err(err).Str("cache-key", key).Msg("Error storing item in cache")
		}
	})
	if errors.Is(err, errJobQueueFull) {
		log.Error().Int("buffer-size", c.config.MaxQueueBufferSize).
			Msg("Failed to store item in cache: job queue full")
		return ErrRedisJobQueueFull
	}
	return err
}

// Delete deletes a key from Redis.
func (c *redisClient) Delete(ctx context.Context, key string) error {
	return c.Del(ctx, key).Err()
}

// Keys returns a slice of cache keys.
func (c *redisClient) Keys(ctx context.Context, prefix string) []string {
	var keys []string
	iter := c.Scan(ctx, 0, prefix+"*", 0).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		panic(err)
	}
	return keys
}

// Stop client and release resources.
func (c *redisClient) Stop() {
	c.queue.stop()
	if err := c.Close(); err != nil {
		log.Error().Err(err).Msg("Failed to stop the redis client.")
	}
}

// Purge purges keys matching the specified pattern from the cache. If the pattern is
// empty, all keys will be removed from the cache, similar to a `Flush`.
func (c *redisClient) Purge(ctx context.Context, pattern string) error {
	iter := c.Scan(ctx, 0, pattern, 0).Iterator()

	for iter.Next(ctx) {
		key := iter.Val()
		if len(key) > 0 {
			// TODO: evaluate non-blocking Unlink.
			if err := c.Del(ctx, key).Err(); err != nil {
				return err
			}
		}
	}

	return iter.Err()
}

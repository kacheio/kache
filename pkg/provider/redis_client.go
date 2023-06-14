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
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
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
}

// redisClient wraps a Redis Universal Client.
type redisClient struct {
	redis.UniversalClient

	// config is the configuration of the client.
	config RedisClientConfig
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
		// todo: logger, add key
		log.Error().Err(err).Msg("failed to get item from redis")
		return nil
	}
	return []byte(res)
}

// Store stores a key and value into Redis.
func (c *redisClient) Store(key string, value []byte, ttl time.Duration) error {
	// TODO: Store async
	_, err := c.Set(context.Background(), key, value, ttl).Result()
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
	if err := c.Close(); err != nil {
		log.Error().Err(err).Msg("Failed to stop the redis client.")
	}
}

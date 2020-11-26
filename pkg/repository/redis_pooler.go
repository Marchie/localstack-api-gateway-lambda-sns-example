package repository

import (
	"context"
	"github.com/gomodule/redigo/redis"
)

type RedisPooler interface {
	Get() redis.Conn
	GetContext(ctx context.Context) (redis.Conn, error)
	Stats() redis.PoolStats
	ActiveCount() int
	IdleCount() int
	Close() error
}

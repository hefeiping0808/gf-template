// Package cache 提供基于 GoFrame Redis 适配器的缓存封装。
package cache

import (
	"context"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

// Set 写入 Redis 缓存；错误由调用方决定是否降级到数据库。
func Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	var err error
	if ttl > 0 {
		seconds := int64(ttl.Seconds())
		if seconds < 1 {
			seconds = 1
		}
		err = g.Redis().SetEX(ctx, key, value, seconds)
	} else {
		_, err = g.Redis().Set(ctx, key, value)
	}
	return err
}

// Get 读取 Redis 缓存，命中情况由业务自行解释。
func Get(ctx context.Context, key string) (any, error) {
	value, err := g.Redis().Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return value.Interface(), nil
}

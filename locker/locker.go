package locker

import (
	"context"
	"time"
)

type Locker interface {
	// Lock 加锁
	Lock(ctx context.Context) error
	// Unlock 解锁
	Unlock(ctx context.Context) error
	// Exist 锁是否存在
	Exist(ctx context.Context) (exist bool, canUnlock bool, err error)
	// ExpireDuration 过期时间段
	ExpireDuration() time.Duration
	// Extend 续期
	Extend(ctx context.Context, deadline time.Time) error
}

func NewLocker(name string, timeout time.Duration) Locker {
	return &locker{
		name:    name,
		secret:  getClusterIp(),
		timeout: timeout,
	}
}

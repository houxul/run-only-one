package locker

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	redisHost         = "127.0.0.1"
	redisPort         = "6379"
	redisPassword     = ""
	globalRedisClient redis.UniversalClient
)

func init() {
	opts := &redis.UniversalOptions{
		Addrs:    []string{net.JoinHostPort(redisHost, redisPort)},
		Password: redisPassword,
	}
	globalRedisClient = redis.NewUniversalClient(opts)
}

func getClusterIp() string {
	return time.Now().String()
}

type locker struct {
	name     string
	secret   string
	timeout  time.Duration
	deadline int64 // 保存time.UnixNano()，表示锁的到期时间
}

func (l *locker) Lock(ctx context.Context) error {
	isSet, err := globalRedisClient.SetNX(ctx, l.name, l.secret, l.timeout).Result()
	if err != nil {
		return err
	}
	if !isSet {
		return errors.New("lock exist")
	}

	deadline := time.Now().Add(l.timeout).UnixNano()
	atomic.StoreInt64(&l.deadline, deadline)
	return nil
}

func (l *locker) Unlock(ctx context.Context) error {
	s, err := globalRedisClient.Get(ctx, l.name).Result()
	if err == redis.Nil {
		return errors.New("lock not found")
	} else if err != nil {
		return err
	}

	if s != l.secret {
		return errors.New("lock " + l.name + " is held by others")
	}

	_, err = globalRedisClient.Del(ctx, l.name).Result()
	if err != nil {
		return err
	}

	atomic.StoreInt64(&l.deadline, 0)
	return nil
}

func (l *locker) Exist(ctx context.Context) (exist bool, canUnlock bool, err error) {
	s, err := globalRedisClient.Get(ctx, l.name).Result()
	if err == redis.Nil {
		return false, false, nil
	} else if err != nil {
		return false, false, err
	}
	return true, s == l.secret, nil
}

func (l *locker) ExpireDuration() time.Duration {
	deadline := atomic.LoadInt64(&l.deadline)
	return time.Duration(deadline - time.Now().UnixNano())
}

func (l *locker) Extend(ctx context.Context, deadline time.Time) error {
	s, err := globalRedisClient.Get(ctx, l.name).Result()
	if err != nil {
		return err
	} else if s != l.secret {
		return errors.New("lock not found")
	}

	_, err = globalRedisClient.ExpireAt(ctx, l.name, deadline).Result()
	if err != nil {
		return err
	}

	atomic.StoreInt64(&l.deadline, deadline.UnixNano())
	return nil
}

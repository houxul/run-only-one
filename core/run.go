package core

import (
	"context"
	"log"
	"time"

	"github.com/pkg/errors"

	. "run-only-one/locker"
)

const (
	// renewDeadline 加锁时间
	leaseDuration = 60 * time.Second
	// renewDeadline 锁续期阈值
	renewDeadline = 15 * time.Second
	// retryPeriod 重试时间
	retryPeriod = 5 * time.Second
)

type RunFunc func(context.Context)

func Run(name string, runFunc RunFunc) func() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	lock := NewLocker(name, leaseDuration)

	go func(ctx context.Context, lock Locker, rf RunFunc) {
		var runFuncCancel context.CancelFunc
		for {
			select {
			case <-ctx.Done():
				return
			default:
				ok, err := captureLock(ctx, lock)
				if err != nil {
					log.Println("invoke grabLock", err)
					if runFuncCancel != nil {
						runFuncCancel()
						runFuncCancel = nil
					}
				} else if ok && runFuncCancel == nil {
					runFuncCtx, ctxCancel := context.WithCancel(ctx)
					runFuncCancel = ctxCancel
					go rf(runFuncCtx)
				}
				time.Sleep(retryPeriod)
			}
		}
	}(ctx, lock, runFunc)

	cancelFunc := func() {
		log.Println("cancel function is called")
		exist, canUnlock, err := lock.Exist(ctx)
		if err != nil {
			log.Println("invoke lock.Exist fail", err)
		}

		if exist && canUnlock {
			lock.Unlock(ctx)
		}

		cancel()
	}
	return cancelFunc
}

func captureLock(ctx context.Context, lock Locker) (bool, error) {
	exist, canUnlock, err := lock.Exist(ctx)
	if err != nil {
		return false, errors.Wrap(err, "invoke lock.Exist fail")
	}
	if exist {
		if canUnlock && lock.ExpireDuration() < renewDeadline {
			if err := lock.Extend(ctx, time.Now().Add(leaseDuration)); err != nil {
				return false, errors.Wrap(err, "invoke lock.Extend fail")
			}
		}
		return canUnlock, nil
	}

	if err := lock.Lock(ctx); err != nil {
		return false, errors.Wrap(err, "invoke lock.Lock fail")
	}

	return true, nil
}

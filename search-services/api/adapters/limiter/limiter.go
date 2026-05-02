package limiter

import (
	"context"
	"time"
)

type Limiter struct {
	leakyBucketCh chan struct{}
}

func NewLimiter(ctx context.Context, limit int, period time.Duration) *Limiter {
	limiter := &Limiter{
		leakyBucketCh: make(chan struct{}, limit),
	}
	leakInterval := period.Nanoseconds() / int64(limit)
	go limiter.Start(ctx, time.Duration(leakInterval))
	return limiter
}

func (v *Limiter) Start(ctx context.Context, interval time.Duration) {
	timer := time.NewTicker(interval)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			select {
			case <-v.leakyBucketCh:
			default:
			}
		}
	}
}

func (v *Limiter) Allow() bool {
	select {
	case v.leakyBucketCh <- struct{}{}:
		return true
	default:
		return false
	}
}

func (v *Limiter) Wait(ctx context.Context) error {
	select {
	case v.leakyBucketCh <- struct{}{}:
		return nil
	default:
	}
	ticker := time.NewTicker(time.Millisecond * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			select {
			case v.leakyBucketCh <- struct{}{}:
				return nil
			default:

			}
		}
	}
}

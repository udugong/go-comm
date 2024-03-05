package retryable

import (
	"context"
	"fmt"
	"time"

	"github.com/udugong/go-comm"
)

// IntervalService 有间隔的重试服务
type IntervalService[T any] struct {
	svc comm.Sender[T]
	// 最大重试次数
	retryMax int
	// 重试间隔
	intervalFunc func() time.Duration
}

func NewIntervalService[T any](svc comm.Sender[T], retryMax int, intervalFn func() time.Duration) *IntervalService[T] {
	return &IntervalService[T]{
		svc:          svc,
		retryMax:     retryMax,
		intervalFunc: intervalFn,
	}
}

func (s *IntervalService[T]) Send(ctx context.Context, biz string, args T, to ...string) error {
	var err error
	timer := time.NewTimer(s.intervalFunc())
	defer timer.Stop()
	for i := 0; i < s.retryMax; {
		err = s.svc.Send(ctx, biz, args, to...)
		if err == nil {
			return nil
		}
		i++
		if i >= s.retryMax {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			timer.Reset(s.intervalFunc())
		}
	}
	return fmt.Errorf("重试 %d 次都失败了; err: %w", s.retryMax, err)
}

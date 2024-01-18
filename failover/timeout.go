package failover

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/udugong/go-comm"
)

// TimeoutService 连续超时故障转移
type TimeoutService[T any] struct {
	svcs []comm.Sender[T]

	// 使用该索引的服务
	idx int32

	// 连续超时次数
	cnt int32

	// 连续超时次数阈值
	threshold int32

	// 重置索引函数
	resetIdxFunc func(ctx context.Context, old *int32)
}

func NewTimeoutService[T any](svcs []comm.Sender[T], threshold int32, opts ...Option[T]) *TimeoutService[T] {
	res := &TimeoutService[T]{
		svcs:      svcs,
		threshold: threshold,
	}
	for _, opt := range opts {
		opt.apply(res)
	}
	return res
}

type Option[T any] interface {
	apply(*TimeoutService[T])
}

type optionFunc[T any] func(*TimeoutService[T])

func (f optionFunc[T]) apply(m *TimeoutService[T]) {
	f(m)
}

func WithResetIdxFunc[T any](fn func(context.Context, *int32)) Option[T] {
	return optionFunc[T](func(t *TimeoutService[T]) {
		t.resetIdxFunc = fn
	})
}

func (t *TimeoutService[T]) Send(ctx context.Context, tpl string, args T, to ...string) error {
	cnt := atomic.LoadInt32(&t.cnt)
	idx := atomic.LoadInt32(&t.idx)
	if cnt >= t.threshold {
		newIdx := (idx + 1) % int32(len(t.svcs))
		if atomic.CompareAndSwapInt32(&t.idx, idx, newIdx) {
			// 重新累计连续的超时次数
			atomic.StoreInt32(&t.cnt, 0)
		}
		idx = newIdx
		if t.resetIdxFunc != nil {
			go t.resetIdxFunc(context.Background(), &t.idx)
		}
	}
	svc := t.svcs[idx]
	err := svc.Send(ctx, tpl, args, to...)
	switch {
	case err == nil:
		// 重新累计连续的超时次数
		atomic.StoreInt32(&t.cnt, 0)
		return nil
	case errors.Is(err, context.DeadlineExceeded): // 超时
		atomic.AddInt32(&t.cnt, 1)
	default:
	}
	return err
}

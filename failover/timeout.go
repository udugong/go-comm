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

	// 设置索引函数
	// 切换了服务后触发
	setIdxFunc func(ctx context.Context, old *int32)
}

func NewTimeoutService[T any](svcs []comm.Sender[T], threshold int32, opts ...TimeoutOption[T]) *TimeoutService[T] {
	res := &TimeoutService[T]{
		svcs:      svcs,
		threshold: threshold,
	}
	for _, opt := range opts {
		opt.apply(res)
	}
	return res
}

type TimeoutOption[T any] interface {
	apply(*TimeoutService[T])
}

type timeoutOptionFunc[T any] func(*TimeoutService[T])

func (f timeoutOptionFunc[T]) apply(svc *TimeoutService[T]) {
	f(svc)
}

func WithSetIdxFunc[T any](fn func(context.Context, *int32)) TimeoutOption[T] {
	return timeoutOptionFunc[T](func(t *TimeoutService[T]) {
		t.setIdxFunc = fn
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
		if t.setIdxFunc != nil {
			go t.setIdxFunc(context.Background(), &t.idx)
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

// GetCurrentServiceIndex 获取当前使用服务的索引
func (t *TimeoutService[T]) GetCurrentServiceIndex() int32 {
	return atomic.LoadInt32(&t.idx)
}

// SetCurrentServiceIndex 根据索引设置使用的服务
// 如果 idx < 0 或者 idx 超过 svcs 的索引范围将统一把 idx 设为 0
func (t *TimeoutService[T]) SetCurrentServiceIndex(idx int32) {
	if idx < 0 || int(idx) >= len(t.svcs) {
		idx = 0
	}
	atomic.StoreInt32(&t.idx, idx)
}

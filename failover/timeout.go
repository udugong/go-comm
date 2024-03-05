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
	idx atomic.Uint32

	// 连续超时次数
	cnt atomic.Uint32

	// 连续超时次数阈值
	threshold uint32

	// 设置索引函数
	// 切换了服务后额外开启一个 goroutine 执行
	setIdxFunc func(ctx context.Context, idx *atomic.Uint32)
}

func NewTimeoutService[T any](svcs []comm.Sender[T], threshold uint32, opts ...TimeoutOption[T]) *TimeoutService[T] {
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

func WithSetIdxFunc[T any](fn func(context.Context, *atomic.Uint32)) TimeoutOption[T] {
	return timeoutOptionFunc[T](func(svc *TimeoutService[T]) {
		svc.setIdxFunc = fn
	})
}

func (t *TimeoutService[T]) Send(ctx context.Context, biz string, args T, to ...string) error {
	cnt := t.cnt.Load()
	idx := t.idx.Load()
	if cnt >= t.threshold {
		newIdx := (idx + 1) % uint32(len(t.svcs))
		if t.idx.CompareAndSwap(idx, newIdx) {
			t.cnt.Store(0)
		}
		idx = newIdx
		if t.setIdxFunc != nil {
			go t.setIdxFunc(context.Background(), &t.idx)
		}
	}
	svc := t.svcs[idx]
	err := svc.Send(ctx, biz, args, to...)
	switch {
	case err == nil:
		// 重新累计连续的超时次数
		t.cnt.Store(0)
		return nil
	case errors.Is(err, context.DeadlineExceeded): // 超时
		t.cnt.Add(1)
	default:
	}
	return err
}

// GetCurrentServiceIndex 获取当前使用服务的索引
func (t *TimeoutService[T]) GetCurrentServiceIndex() uint32 {
	return t.idx.Load()
}

// SetCurrentServiceIndex 根据索引设置使用的服务
// idx 超过 svcs 的索引范围将统一把 idx 设为 0
func (t *TimeoutService[T]) SetCurrentServiceIndex(idx uint32) {
	if int(idx) >= len(t.svcs) {
		idx = 0
	}
	t.idx.Store(idx)
}

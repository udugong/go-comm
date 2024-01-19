package failover

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/udugong/go-comm"
)

// Service 出现错误时故障转移
type Service[T any] struct {
	svcs []comm.Sender[T]
	idx  int32
}

func NewService[T any](svcs []comm.Sender[T]) *Service[T] {
	return &Service[T]{
		svcs: svcs,
	}
}

func (s *Service[T]) Send(ctx context.Context, tpl string, args T, to ...string) error {
	idx := atomic.LoadInt32(&s.idx)
	length := int32(len(s.svcs))
	for i := idx; i < idx+length; i++ {
		svc := s.svcs[int(i%length)]
		err := svc.Send(ctx, tpl, args, to...)
		if err == nil {
			return nil
		}
		atomic.AddInt32(&s.idx, 1)
		switch {
		case errors.Is(err, context.DeadlineExceeded),
			errors.Is(err, context.Canceled):
			return err
		default:
		}
	}
	if atomic.LoadInt32(&s.idx) >= length {
		atomic.StoreInt32(&s.idx, 0)
	}
	return ErrAllServiceFailed
}

// GetCurrentServiceIndex 获取当前使用服务的索引
func (s *Service[T]) GetCurrentServiceIndex() int32 {
	return atomic.LoadInt32(&s.idx)
}

// SetCurrentServiceIndex 根据索引设置使用的服务
// 如果 idx < 0 或者 idx 超过 svcs 的索引范围将统一把 idx 设为 0
func (s *Service[T]) SetCurrentServiceIndex(idx int32) {
	if idx < 0 || int(idx) >= len(s.svcs) {
		idx = 0
	}
	atomic.StoreInt32(&s.idx, idx)
}

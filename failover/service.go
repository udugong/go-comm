package failover

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"

	"github.com/udugong/go-comm"
)

// Service 出现错误时故障转移
type Service[T any] struct {
	svcs   []comm.Sender[T]
	idx    atomic.Uint32
	logger *slog.Logger
}

func NewService[T any](svcs []comm.Sender[T], opts ...Option[T]) *Service[T] {
	s := &Service[T]{
		svcs:   svcs,
		logger: slog.Default(),
	}
	return s.WithOptions(opts...)
}

type Option[T any] interface {
	apply(*Service[T])
}

type optionFunc[T any] func(*Service[T])

func (f optionFunc[T]) apply(svc *Service[T]) {
	f(svc)
}

func WithLogger[T any](logger *slog.Logger) Option[T] {
	return optionFunc[T](func(svc *Service[T]) {
		svc.logger = logger
	})
}

func (s *Service[T]) Send(ctx context.Context, tpl string, args T, to ...string) error {
	idx := s.idx.Load()
	length := uint32(len(s.svcs))
	for i := idx; i < idx+length; i++ {
		svc := s.svcs[i%length]
		err := svc.Send(ctx, tpl, args, to...)
		if err == nil {
			return nil
		}
		s.idx.Add(1)
		switch {
		case errors.Is(err, context.DeadlineExceeded),
			errors.Is(err, context.Canceled):
			return err
		default:
			s.logger.LogAttrs(ctx, slog.LevelError, "发送失败准备使用下一个服务重试",
				slog.Uint64("svc_idx", uint64(i%length)), slog.Any("err", err))
		}
	}
	return ErrAllServiceFailed
}

// GetCurrentServiceIndex 获取当前使用服务的索引
func (s *Service[T]) GetCurrentServiceIndex() uint32 {
	return s.idx.Load()
}

// SetCurrentServiceIndex 根据索引设置使用的服务
// idx 超过 svcs 的索引范围将统一把 idx 设为 0
func (s *Service[T]) SetCurrentServiceIndex(idx uint32) {
	if int(idx) >= len(s.svcs) {
		idx = 0
	}
	s.idx.Store(idx)
}

func (s *Service[T]) WithOptions(opts ...Option[T]) *Service[T] {
	for _, opt := range opts {
		opt.apply(s)
	}
	return s
}

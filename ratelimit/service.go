package ratelimit

import (
	"context"
	"fmt"

	"github.com/udugong/limiter"

	"github.com/udugong/go-comm"
)

// Service 限流服务
type Service[T any] struct {
	svc comm.Sender[T]

	// 限流的key
	limitKey string

	// 限流器
	limiter limiter.Limiter
}

func NewService[T any](svc comm.Sender[T], limitKey string, limiter limiter.Limiter) *Service[T] {
	return &Service[T]{
		svc:      svc,
		limitKey: limitKey,
		limiter:  limiter,
	}
}

func (s *Service[T]) Send(ctx context.Context, tpl string, args T, to ...string) error {
	limited, err := s.limiter.Limit(ctx, s.limitKey)
	if err != nil {
		return fmt.Errorf("sender 服务判断限流器出现问题; err: %w", err)
	}
	if limited {
		return ErrLimited
	}
	return s.svc.Send(ctx, tpl, args, to...)
}

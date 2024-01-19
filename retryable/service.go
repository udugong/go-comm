package retryable

import (
	"context"
	"fmt"

	"github.com/udugong/go-comm"
)

// Service 重试服务
type Service[T any] struct {
	svc comm.Sender[T]
	// 最大重试次数
	retryMax int
}

func NewService[T any](svc comm.Sender[T], retryMax int) *Service[T] {
	return &Service[T]{
		svc:      svc,
		retryMax: retryMax,
	}
}

func (s *Service[T]) Send(ctx context.Context, tpl string, args T, to ...string) error {
	var err error
	for i := 0; i < s.retryMax; {
		err = s.svc.Send(ctx, tpl, args, to...)
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
		default:
		}
	}
	return fmt.Errorf("重试 %d 次都失败了; err: %w", s.retryMax, err)
}

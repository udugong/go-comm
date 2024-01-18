package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestService_Send(t *testing.T) {
	testSvc := &testService[int]{}
	testLimiter := &testLimiter{}
	svc := NewService[int](testSvc, "ses", testLimiter)
	type testCase[T any] struct {
		name     string
		ctx      context.Context
		limited  bool
		limitErr error
		wantErr  error
	}
	tests := []testCase[int]{
		{
			name: "normal",
			ctx:  context.Background(),
		},
		{
			name:    "limited",
			ctx:     context.Background(),
			limited: true,
			wantErr: ErrLimited,
		},
		{
			name:     "limiter_error",
			ctx:      context.Background(),
			limitErr: errors.New("模拟限流器错误"),
			wantErr: fmt.Errorf("sender 服务判断限流器出现问题; err: %w",
				errors.New("模拟限流器错误")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testLimiter.limited = tt.limited
			testLimiter.err = tt.limitErr
			err := svc.Send(tt.ctx, "", 0, "")
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

type testService[T any] struct {
	err error
}

func (svc *testService[T]) Send(_ context.Context, _ string, _ T, _ ...string) error {
	return svc.err
}

type testLimiter struct {
	limited bool
	err     error
}

func (l *testLimiter) Limit(ctx context.Context, key string) (bool, error) {
	return l.limited, l.err
}

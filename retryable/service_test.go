package retryable

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestService_Send(t *testing.T) {
	retryMax := 3
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()
	testSvc := &testService[int]{}
	svc := NewService[int](testSvc, retryMax, 0)
	type testCase[T any] struct {
		name        string
		ctx         context.Context
		localSvcErr error
		wantErr     error
	}
	tests := []testCase[int]{
		{
			name: "normal",
			ctx:  context.Background(),
		},
		{
			name:        "service_error",
			ctx:         context.Background(),
			localSvcErr: errors.New("模拟服务错误"),
			wantErr:     fmt.Errorf("重试 %d 次都失败了; err: %w", retryMax, errors.New("模拟服务错误")),
		},
		{
			name:        "timeout_error",
			ctx:         timeoutCtx,
			localSvcErr: errors.New("模拟服务错误"),
			wantErr:     context.DeadlineExceeded,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testSvc.err = tt.localSvcErr
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

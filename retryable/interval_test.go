package retryable

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIntervalService_Send(t *testing.T) {
	retryMax := 3
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()
	testSvc := &testService[int]{}
	svc := NewIntervalService[int](testSvc, retryMax,
		func() time.Duration {
			return time.Duration(100+rand.Intn(50)) * time.Millisecond
		})
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

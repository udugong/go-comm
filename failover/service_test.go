package failover

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/udugong/go-comm"
)

func TestService_Send(t *testing.T) {
	type testCase[T any] struct {
		name         string
		localSvc1Err error
		localSvc2Err error
		wantErr      error
		wantIdx      int32
	}
	tests := []testCase[[]string]{
		{
			name:    "normal",
			wantIdx: 0,
		},
		{
			name:         "svc1_error",
			localSvc1Err: errors.New("模拟svc1错误"),
			wantIdx:      1,
		},
		{
			name:         "all_svc_error",
			localSvc1Err: errors.New("模拟svc1错误"),
			localSvc2Err: errors.New("模拟svc2错误"),
			wantErr:      ErrAllServiceFailed,
			wantIdx:      0,
		},
		{
			name:         "context_cancel",
			localSvc1Err: context.Canceled,
			wantErr:      context.Canceled,
			wantIdx:      1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localSvc1 := &testService[[]string]{}
			localSvc2 := &testService[[]string]{}
			svc := NewService[[]string](
				[]comm.Sender[[]string]{
					localSvc1,
					localSvc2,
				},
			)
			localSvc1.err = tt.localSvc1Err
			localSvc2.err = tt.localSvc2Err
			err := svc.Send(context.Background(), "", []string{}, "")
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.wantIdx, svc.idx)
		})
	}
}

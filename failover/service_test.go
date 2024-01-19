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

func TestService_GetCurrentServiceIndex(t *testing.T) {
	svc1 := &testService[int]{}
	svc2 := &testService[int]{}
	svc := NewService[int]([]comm.Sender[int]{svc1, svc2})
	type testCase[T any] struct {
		name string
		want int32
	}
	tests := []testCase[int]{
		{
			name: "normal",
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc.idx = tt.want
			assert.Equal(t, tt.want, svc.GetCurrentServiceIndex())
		})
	}
}

func TestService_SetCurrentServiceIndex(t *testing.T) {
	svc1 := &testService[int]{}
	svc2 := &testService[int]{}
	svc := NewService[int]([]comm.Sender[int]{svc1, svc2})
	type testCase[T any] struct {
		name string
		idx  int32
		want int32
	}
	tests := []testCase[int]{
		{
			name: "normal",
			idx:  1,
			want: 1,
		},
		{
			name: "index_is_less_than_range",
			idx:  -1,
			want: 0,
		},
		{
			name: "index_greater_than_range",
			idx:  2,
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc.SetCurrentServiceIndex(tt.idx)
			assert.Equal(t, tt.want, svc.idx)
		})
	}
}

type testService[T any] struct {
	err error
}

func (svc *testService[T]) Send(_ context.Context, _ string, _ T, _ ...string) error {
	return svc.err
}

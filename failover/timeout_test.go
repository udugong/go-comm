package failover

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/udugong/go-comm"
)

func TestTimeoutService_Send(t *testing.T) {
	type testCase[T any] struct {
		name         string
		localSvc1Err error
		localSvc2Err error
		fn           func(context.Context, *int32)
		interval     time.Duration
		wantIdx      int32
	}
	tests := []testCase[[]string]{
		{
			name:    "normal",
			wantIdx: 0,
		},
		{
			name:         "svc1_timeout",
			localSvc1Err: context.DeadlineExceeded,
			wantIdx:      1,
		},
		{
			name:         "all_svc_timeout",
			localSvc1Err: context.DeadlineExceeded,
			localSvc2Err: context.DeadlineExceeded,
			wantIdx:      0,
		},
		{
			name:         "another_error",
			localSvc1Err: errors.New("另外的错误"),
			wantIdx:      0,
		},
		{
			name:         "svc1_timeout_and_reset_idx",
			localSvc1Err: context.DeadlineExceeded,
			fn:           withResetIdxFunc,
			interval:     150 * time.Millisecond,
			wantIdx:      0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localSvc1 := &testService[[]string]{}
			localSvc2 := &testService[[]string]{}
			svcs := []comm.Sender[[]string]{
				localSvc1,
				localSvc2,
			}
			svc := NewTimeoutService[[]string](
				svcs, 1, WithResetIdxFunc[[]string](tt.fn),
			)
			localSvc1.err = tt.localSvc1Err
			localSvc2.err = tt.localSvc2Err
			for i := 0; i < len(svc.svcs)+1; i++ {
				_ = svc.Send(context.Background(), "", []string{}, "")
			}
			<-time.After(tt.interval)
			assert.Equal(t, tt.wantIdx, svc.idx)
		})
	}
}

type testService[T any] struct {
	err error
}

func (svc *testService[T]) Send(_ context.Context, _ string, _ T, _ ...string) error {
	return svc.err
}

func withResetIdxFunc(ctx context.Context, old *int32) {
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		atomic.StoreInt32(old, 0)
	}
}

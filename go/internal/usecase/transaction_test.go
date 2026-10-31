package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/lib/pq"
)

func TestRetryOnDeadlock(t *testing.T) {
	t.Parallel()

	deadlock := &pq.Error{Code: "40P01"}
	other := errors.New("デッドロック以外のエラー")
	for _, tt := range []struct {
		name     string
		errors   []error
		wantRuns int
		wantErr  error
	}{
		{name: "成功すれば繰り返さない", errors: []error{nil}, wantRuns: 1},
		{name: "ラップされたデッドロックから回復する", errors: []error{fmt.Errorf("書き込みに失敗: %w", deadlock), nil}, wantRuns: 2},
		{name: "上限に達したら最後のエラーを返す", errors: []error{deadlock, deadlock, deadlock}, wantRuns: 3, wantErr: deadlock},
		{name: "ほかのエラーは繰り返さない", errors: []error{other}, wantRuns: 1, wantErr: other},
		{name: "再試行中にほかのエラーが起きたら返す", errors: []error{deadlock, other}, wantRuns: 2, wantErr: other},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runs := 0
			output, err := retryOnDeadlock(t.Context(), func() (int, error) {
				runs++
				if runs > len(tt.errors) {
					t.Fatal("予定より多く再試行されました")
				}
				return runs, tt.errors[runs-1]
			})
			if runs != tt.wantRuns || output != tt.wantRuns || !errors.Is(err, tt.wantErr) {
				t.Errorf("実行回数・出力・エラー = %d / %d / %v、期待値 = %d / %d / %v", runs, output, err, tt.wantRuns, tt.wantRuns, tt.wantErr)
			}
		})
	}
}

func TestRetryOnDeadlock_Cancellation(t *testing.T) {
	t.Parallel()

	for _, beforeStart := range []bool{true, false} {
		name, wantRuns := "待機中の中止で再試行しない", 1
		if beforeStart {
			name, wantRuns = "中止済みなら開始しない", 0
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if beforeStart {
				cancel()
			}
			runs := 0
			_, err := retryOnDeadlock(ctx, func() (int, error) {
				runs++
				cancel()
				return 0, &pq.Error{Code: "40P01"}
			})
			if runs != wantRuns || !errors.Is(err, context.Canceled) {
				t.Errorf("実行回数・エラー = %d / %v、期待値 = %d / context.Canceled", runs, err, wantRuns)
			}
		})
	}
}

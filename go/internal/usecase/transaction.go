package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/lib/pq"
)

const deadlockRetryLimit = 3

// retryOnDeadlockは、デッドロックで中止された処理を取得・認可・検証からやり直す。
// runは失敗したトランザクションをロールバックしてから返すこと。
// 同じ競合を続けて起こさないよう、再試行の間隔を延ばし、回数と待ち時間を制限する
func retryOnDeadlock[T any](ctx context.Context, run func() (T, error)) (T, error) {
	var zero T
	for attempt := 0; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return zero, err
		}

		output, err := run()
		var pqErr *pq.Error
		if !errors.As(err, &pqErr) || pqErr.Code != "40P01" || attempt+1 == deadlockRetryLimit {
			return output, err
		}

		timer := time.NewTimer(time.Duration(attempt+1) * 10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return zero, ctx.Err()
		case <-timer.C:
		}
	}
}

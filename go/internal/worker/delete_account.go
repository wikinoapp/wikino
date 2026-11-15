package worker

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/riverqueue/river"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

const (
	// deleteAccountSnoozeIntervalは、スペースメンバーの行が残っているユーザーの物理削除を
	// 先送りする間隔。行は削除済みのスペースの物理削除が消すため、その完了を待つ。
	deleteAccountSnoozeInterval = time.Hour
	// deleteAccountMaxSnoozesは、物理削除を先送りする回数の上限 (間隔と合わせて3日)。
	// スペースの物理削除はふつう数分で終わるため、超えたら行が消えない原因があるとみなし、
	// エラーとして記録して調べられるようにする。
	deleteAccountMaxSnoozes = 72
)

// DeleteAccountWorkerは論理削除したユーザーを物理削除するジョブを受け取り、UseCaseへ渡す。
type DeleteAccountWorker struct {
	river.WorkerDefaults[dispatcher.DeleteAccountArgs]
	uc *usecase.DeleteAccountUsecase
}

// NewDeleteAccountWorkerはDeleteAccountWorkerを生成する。
func NewDeleteAccountWorker(uc *usecase.DeleteAccountUsecase) *DeleteAccountWorker {
	return &DeleteAccountWorker{uc: uc}
}

// Workは論理削除したユーザーを物理削除する。スペースメンバーの行が残っている間は、
// 上限の回数まで先送りし、超えたらUseCaseのエラーをそのまま返す。
func (w *DeleteAccountWorker) Work(ctx context.Context, job *river.Job[dispatcher.DeleteAccountArgs]) error {
	err := w.uc.Execute(ctx, usecase.DeleteAccountInput{
		UserID: model.UserID(job.Args.UserID),
	})
	if errors.Is(err, usecase.ErrAccountHasSpaceMembers) && snoozeCount(job.Metadata) < deleteAccountMaxSnoozes {
		return river.JobSnooze(deleteAccountSnoozeInterval)
	}
	return err
}

// snoozeCountは、ジョブがこれまでに先送りされた回数を返す。riverはメタデータの "snoozes" に
// 回数を記録する (先送りは試行回数に数えないため、Attemptからは分からない)。
func snoozeCount(metadata []byte) int {
	var m struct {
		Snoozes int `json:"snoozes"`
	}
	if err := json.Unmarshal(metadata, &m); err != nil {
		return 0
	}
	return m.Snoozes
}

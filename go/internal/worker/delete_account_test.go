package worker

import (
	"errors"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

func TestDeleteAccountArgs(t *testing.T) {
	t.Parallel()

	args := dispatcher.DeleteAccountArgs{}
	if args.Kind() != "delete_account" {
		t.Errorf("Kind() = %s、期待値 = delete_account", args.Kind())
	}
	opts := args.InsertOpts()
	if opts.Queue != river.QueueDefault {
		t.Errorf("Queue = %s、期待値 = %s", opts.Queue, river.QueueDefault)
	}
	if opts.MaxAttempts != 5 {
		t.Errorf("MaxAttempts = %d、期待値 = 5", opts.MaxAttempts)
	}
}

// newDeleteAccountJobは、これまでの先送りの回数をメタデータに持つジョブを組み立てる。
func newDeleteAccountJob(userID model.UserID, metadata string) *river.Job[dispatcher.DeleteAccountArgs] {
	return &river.Job[dispatcher.DeleteAccountArgs]{
		JobRow: &rivertype.JobRow{Metadata: []byte(metadata)},
		Args:   dispatcher.DeleteAccountArgs{UserID: string(userID)},
	}
}

func TestDeleteAccountWorker_Work_スペースメンバーの行が残る間は先送りする(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	queries := query.New(db)
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("worker-delete-account@example.com").
		WithAtname("worker_delete_account").
		Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier("worker-delete-account").Build()
	testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(userID).Build()
	if _, err := db.ExecContext(t.Context(), `UPDATE users SET discarded_at = NOW() WHERE id = $1`, string(userID)); err != nil {
		t.Fatalf("ユーザーの論理削除に失敗: %v", err)
	}
	w := NewDeleteAccountWorker(usecase.NewDeleteAccountUsecase(
		db,
		repository.NewUserRepository(queries),
		repository.NewUserSessionRepository(queries),
		repository.NewUserPasswordRepository(queries),
		repository.NewUserTwoFactorAuthRepository(queries),
		repository.NewSpaceMemberRepository(queries),
	))

	t.Run("上限の回数までは先送りする", func(t *testing.T) {
		for _, metadata := range []string{`{}`, `{"snoozes": 71}`} {
			err := w.Work(t.Context(), newDeleteAccountJob(userID, metadata))
			var snooze *rivertype.JobSnoozeError
			if !errors.As(err, &snooze) {
				t.Errorf("メタデータ%sでのWork()のエラー = %v、先送りを期待", metadata, err)
				continue
			}
			if snooze.Duration != deleteAccountSnoozeInterval {
				t.Errorf("先送りの間隔 = %v、期待値 = %v", snooze.Duration, deleteAccountSnoozeInterval)
			}
		}
	})

	t.Run("上限を超えたらエラーとして記録する", func(t *testing.T) {
		err := w.Work(t.Context(), newDeleteAccountJob(userID, `{"snoozes": 72}`))
		if !errors.Is(err, usecase.ErrAccountHasSpaceMembers) {
			t.Errorf("Work()のエラー = %v、ErrAccountHasSpaceMembersを期待", err)
		}
	})
}

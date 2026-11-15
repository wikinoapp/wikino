package usecase

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// accountDeletionCallbackInserterは、ジョブを投入し終わる前の二重送信を再現する。
type accountDeletionCallbackInserter struct {
	call func() error
}

func (i accountDeletionCallbackInserter) Insert(context.Context, river.JobArgs, *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return nil, i.call()
}

func TestDiscardAccountUsecase_Execute_二重送信に成功を返さず投入失敗で復元する(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(t.Context(), i18n.LangJa)
	db := testutil.GetTestDB()
	user := seedDiscardAccountUser(t, db, "duplicate-enqueue-failure")
	input := DiscardAccountInput{User: user, Atname: user.Atname}
	var uc *DiscardAccountUsecase
	var duplicateErr error
	insertions := 0
	uc = newDiscardAccountUsecase(db, accountDeletionCallbackInserter{call: func() error {
		insertions++
		duplicateErr = uc.Execute(ctx, input)
		return errors.New("ジョブの投入に失敗")
	}})
	if err := uc.Execute(ctx, input); err == nil {
		t.Fatal("先行要求がジョブ投入の失敗を返さなかった")
	}
	if ae := model.AsAppError(duplicateErr); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
		t.Errorf("二重送信のエラー = %v、ResourceNotFoundを期待", duplicateErr)
	}
	if insertions != 1 {
		t.Errorf("ジョブの投入試行 = %d回、期待値 = 1回", insertions)
	}
	discarded, sessions := userState(t, db, user.ID)
	if discarded || sessions != 0 {
		t.Errorf("ユーザーの状態 = 論理削除%v・セッション%d件、期待値 = 論理削除false・セッション0件", discarded, sessions)
	}
}

func TestDiscardAccountUsecase_Execute_ロック待ちの間にスペースへ参加したら削除しない(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(i18n.SetLocale(t.Context(), i18n.LangJa), 10*time.Second)
	defer cancel()
	db := testutil.GetTestDB()
	user := seedDiscardAccountUser(t, db, "waiting-for-space")
	tx, pid := lockAccountUser(t, ctx, db, user.ID)
	inserter := &mockJobInserter{}
	result := make(chan error, 1)
	go func() {
		result <- newDiscardAccountUsecase(db, inserter).Execute(ctx, DiscardAccountInput{User: user, Atname: user.Atname})
	}()
	waitForAccountUserLock(t, ctx, db, pid, result)

	// スペース作成側がユーザー行をロックしたまま、スペースと参加行をコミットする。
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("account-waiting-for-space").Build()
	testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(spaceID).WithUserID(user.ID).Build()
	if err := tx.Commit(); err != nil {
		t.Fatalf("スペースの作成のコミットに失敗: %v", err)
	}

	err := <-result
	if ve := model.AsValidationError(err); ve == nil || len(ve.Global) == 0 {
		t.Fatalf("Execute()のエラー = %v、参加中のスペースがあることのValidationErrorを期待", err)
	}
	discarded, sessions := userState(t, db, user.ID)
	if discarded || sessions != 2 {
		t.Errorf("ユーザーの状態 = 論理削除%v・セッション%d件、期待値 = 論理削除false・セッション2件", discarded, sessions)
	}
	if inserter.called {
		t.Error("参加中のユーザーの物理削除のジョブが投入された")
	}
}

func TestCreateSpaceUsecase_Execute_ロック待ちの間に削除されたユーザーを参加させない(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(i18n.SetLocale(t.Context(), i18n.LangJa), 10*time.Second)
	defer cancel()
	db := testutil.GetTestDB()
	user := seedDiscardAccountUser(t, db, "waiting-for-deletion")
	tx, pid := lockAccountUser(t, ctx, db, user.ID)
	uc, spaceRepo, _ := newCreateSpaceUsecase()
	const identifier = "account-wait-delete"
	result := make(chan error, 1)
	go func() {
		_, err := uc.Execute(ctx, CreateSpaceInput{UserID: user.ID, Identifier: identifier, Name: "作成されないスペース"})
		result <- err
	}()
	waitForAccountUserLock(t, ctx, db, pid, result)

	// アカウント削除側が先にロックを取った場合、待っていた作成側は削除後の状態を読む。
	queries := query.New(tx)
	if _, err := repository.NewUserRepository(queries).Discard(ctx, user.ID, time.Now()); err != nil {
		t.Fatalf("ユーザーの論理削除に失敗: %v", err)
	}
	if err := repository.NewUserSessionRepository(queries).DeleteByUserID(ctx, user.ID); err != nil {
		t.Fatalf("セッションの削除に失敗: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("アカウントの削除のコミットに失敗: %v", err)
	}

	err := <-result
	if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
		t.Fatalf("Execute()のエラー = %v、ResourceNotFoundを期待", err)
	}
	if exists, err := spaceRepo.ExistsByIdentifier(ctx, identifier); err != nil || exists {
		t.Errorf("スペースの存在 = %v、エラー = %v、期待値 = false, nil", exists, err)
	}
	var members int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM space_members WHERE user_id = $1`, string(user.ID)).Scan(&members); err != nil {
		t.Fatalf("スペースメンバーの確認に失敗: %v", err)
	}
	if members != 0 {
		t.Errorf("論理削除後の参加行 = %d件、期待値 = 0件", members)
	}
}

// lockAccountUserは、スペース作成・アカウント削除と同じユーザー行をロックし、接続のPIDも返す。
func lockAccountUser(t *testing.T, ctx context.Context, db *sql.DB, userID model.UserID) (*sql.Tx, int) {
	t.Helper()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("トランザクションの開始に失敗: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })
	locked, err := repository.NewUserRepository(query.New(tx)).LockActive(ctx, userID)
	if err != nil || !locked {
		t.Fatalf("ユーザーのロック = %v、エラー = %v、期待値 = true, nil", locked, err)
	}
	var pid int
	if err := tx.QueryRowContext(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
		t.Fatalf("ロックした接続のPID取得に失敗: %v", err)
	}
	return tx, pid
}

// waitForAccountUserLockは、対象のユーザー行のロック待ちをDBで確認する。
// 固定の時間だけ待つと、並行要求が検証を終える前にテストが進んでしまうためである。
func waitForAccountUserLock(t *testing.T, ctx context.Context, db *sql.DB, pid int, result <-chan error) {
	t.Helper()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var blocked bool
		if err := db.QueryRowContext(ctx, `SELECT EXISTS (
			SELECT 1 FROM pg_stat_activity
			WHERE datname = current_database()
			  AND $1 = ANY(pg_blocking_pids(pid))
			  AND query LIKE '%LockActiveUser%'
		)`, pid).Scan(&blocked); err != nil {
			t.Fatalf("ユーザー行のロック待ちの確認に失敗: %v", err)
		}
		if blocked {
			return
		}
		select {
		case err := <-result:
			t.Fatalf("ユーザー行のロックを待たずに要求が終了: %v", err)
		case <-ctx.Done():
			t.Fatalf("ユーザー行のロックを待たなかった: %v", ctx.Err())
		case <-ticker.C:
		}
	}
}

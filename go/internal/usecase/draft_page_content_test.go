package usecase

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// findOrCreateDraftPageResultは、別のゴルーチンで実行した下書きの取得・作成の結果
type findOrCreateDraftPageResult struct {
	draftPage *model.DraftPage
	err       error
}

// TestFindOrCreateDraftPage_ConcurrentCreateは、取得した後に他のトランザクションが同じ下書きを
// 作成してコミットしても、一意制約の違反でトランザクションを中断させず、その下書きを返すことを確かめる
func TestFindOrCreateDraftPage_ConcurrentCreate(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	draftPageRepo := repository.NewDraftPageRepository(query.New(db))

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("find-or-create-draft-concurrent").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("find-or-create-draft-concurrent@example.com").
		WithAtname("findorcreatedraftconc").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithName("General").
		Build()
	pageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()

	// 先に下書きを作成し、コミットせずに保持するトランザクション
	otherTx, err := db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatalf("他のトランザクションの開始: %v", err)
	}
	t.Cleanup(func() { _ = otherTx.Rollback() })
	var otherPID int
	if err := otherTx.QueryRowContext(t.Context(), "SELECT pg_backend_pid()").Scan(&otherPID); err != nil {
		t.Fatalf("他のトランザクションの接続のPID取得: %v", err)
	}
	otherDraftPageID := testutil.NewDraftPageBuilder(t, otherTx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithBody("他のトランザクションの下書き").
		Build()

	tx, err := db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatalf("トランザクションの開始: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })
	txDraftPageRepo := draftPageRepo.WithTx(tx)

	// 未コミットの下書きは見えないため作成に進み、他のトランザクションの終了を待つ
	resultCh := make(chan findOrCreateDraftPageResult, 1)
	go func() {
		draftPage, err := findOrCreateDraftPage(t.Context(), txDraftPageRepo, saveDraftPageContentInput{
			SpaceID:       spaceID,
			PageID:        pageID,
			SpaceMemberID: spaceMemberID,
			TopicID:       topicID,
		}, time.Now())
		resultCh <- findOrCreateDraftPageResult{draftPage: draftPage, err: err}
	}()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	waitUntilDraftPageCreateBlockedBy(t, ctx, db, otherPID, resultCh)

	if err := otherTx.Commit(); err != nil {
		t.Fatalf("他のトランザクションのコミット: %v", err)
	}

	result := <-resultCh
	if result.err != nil {
		t.Fatalf("findOrCreateDraftPage()のエラー = %v、期待値 = nil", result.err)
	}
	if result.draftPage == nil || result.draftPage.ID != otherDraftPageID {
		t.Fatalf("findOrCreateDraftPage() = %+v、期待値 = ID %v の下書き", result.draftPage, otherDraftPageID)
	}

	// 競合の後もトランザクションは中断されておらず、続けて下書きを更新してコミットできる
	if _, err := txDraftPageRepo.Update(t.Context(), repository.UpdateDraftPageInput{
		ID:         result.draftPage.ID,
		SpaceID:    spaceID,
		TopicID:    topicID,
		Body:       "更新した本文",
		ModifiedAt: time.Now(),
	}); err != nil {
		t.Fatalf("競合後の下書きの更新: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("トランザクションのコミット: %v", err)
	}
}

// waitUntilDraftPageCreateBlockedByは、下書きの作成がotherPIDの接続のロックを待つまで待機する。
// 実行順に依存するsleepでは競合が成立しないことがあるため、ブロック状態をDBで確認する。
// 待つ前に取得・作成が終わった場合と、ctxが期限切れになった場合はテストを失敗させる
func waitUntilDraftPageCreateBlockedBy(t *testing.T, ctx context.Context, db *sql.DB, otherPID int, resultCh <-chan findOrCreateDraftPageResult) {
	t.Helper()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var blocked bool
		err := db.QueryRowContext(ctx, `SELECT EXISTS (
			SELECT 1 FROM pg_stat_activity
			WHERE datname = current_database()
			  AND $1 = ANY(pg_blocking_pids(pid))
			  AND query LIKE '%INSERT INTO draft_pages%'
		)`, otherPID).Scan(&blocked)
		if err != nil {
			t.Fatalf("下書きの作成のロック待ちの確認: %v", err)
		}
		if blocked {
			return
		}
		select {
		case result := <-resultCh:
			t.Fatalf("競合前に取得・作成が終了: draftPage=%+v, err=%v", result.draftPage, result.err)
		case <-ctx.Done():
			t.Fatalf("下書きの作成が他のトランザクションを待たなかった: %v", ctx.Err())
		case <-ticker.C:
		}
	}
}

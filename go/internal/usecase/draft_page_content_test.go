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

// draftPageSaveResultは、別のゴルーチンで実行した下書きの取得・作成・保存の結果
type draftPageSaveResult struct {
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
	resultCh := make(chan draftPageSaveResult, 1)
	go func() {
		draftPage, err := findOrCreateDraftPage(t.Context(), txDraftPageRepo, saveDraftPageContentInput{
			SpaceID:       spaceID,
			PageID:        pageID,
			SpaceMemberID: spaceMemberID,
			TopicID:       topicID,
		}, time.Now())
		resultCh <- draftPageSaveResult{draftPage: draftPage, err: err}
	}()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	waitUntilDraftPageSaveBlockedBy(t, ctx, db, otherPID, "INSERT INTO draft_pages", resultCh)

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

// waitUntilDraftPageSaveBlockedByは、下書きの保存の途中のqueryNameを含むクエリが、otherPIDの接続の
// ロックを待つまで待機する。実行順に依存するsleepでは競合が成立しないことがあるため、ブロック状態をDBで確認する。
// 待つ前に保存が終わった場合と、ctxが期限切れになった場合はテストを失敗させる
func waitUntilDraftPageSaveBlockedBy(t *testing.T, ctx context.Context, db *sql.DB, otherPID int, queryName string, resultCh <-chan draftPageSaveResult) {
	t.Helper()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var blocked bool
		err := db.QueryRowContext(ctx, `SELECT EXISTS (
			SELECT 1 FROM pg_stat_activity
			WHERE datname = current_database()
			  AND $1 = ANY(pg_blocking_pids(pid))
			  AND query LIKE '%' || $2 || '%'
		)`, otherPID, queryName).Scan(&blocked)
		if err != nil {
			t.Fatalf("下書きの保存のロック待ちの確認: %v", err)
		}
		if blocked {
			return
		}
		select {
		case result := <-resultCh:
			t.Fatalf("競合前に保存が終了: draftPage=%+v, err=%v", result.draftPage, result.err)
		case <-ctx.Done():
			t.Fatalf("下書きの保存の %s が他のトランザクションを待たなかった: %v", queryName, ctx.Err())
		case <-ticker.C:
		}
	}
}

// TestSaveDraftPageContent_TopicFollowsPageは、渡されたトピックや既存の下書きのトピックがページのトピックと
// 食い違っていても、下書きをページのトピックで保存することを確かめる
func TestSaveDraftPageContent_TopicFollowsPage(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	q := query.New(db)
	draftPageRepo := repository.NewDraftPageRepository(q).WithTx(tx)
	pageRepo := repository.NewPageRepository(q).WithTx(tx)
	pageEditorRepo := repository.NewPageEditorRepository(q).WithTx(tx)
	topicRepo := repository.NewTopicRepository(q).WithTx(tx)

	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("save-draft-topic-follows-page").Build()
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("save-draft-topic-follows-page@example.com").
		WithAtname("savedrafttopicfollows").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(spaceID).WithUserID(userID).Build()
	oldTopicID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).WithNumber(1).WithName("Old").Build()
	pageTopicID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).WithNumber(2).WithName("Current").Build()

	tests := []struct {
		name          string
		existingDraft bool
	}{
		{name: "既存の下書きのトピックがページと食い違う", existingDraft: true},
		{name: "下書きが無く、渡されたトピックがページと食い違う", existingDraft: false},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pageID := testutil.NewPageBuilder(t, tx).
				WithSpaceID(spaceID).
				WithTopicID(pageTopicID).
				WithNumber(model.PageNumber(i + 1)).
				WithTitle(tt.name).
				Build()
			if tt.existingDraft {
				testutil.NewDraftPageBuilder(t, tx).
					WithSpaceID(spaceID).
					WithPageID(pageID).
					WithSpaceMemberID(spaceMemberID).
					WithTopicID(oldTopicID).
					Build()
			}

			draftPage, err := saveDraftPageContent(t.Context(), saveDraftPageContentInput{
				SpaceID:          spaceID,
				PageID:           pageID,
				SpaceMemberID:    spaceMemberID,
				TopicID:          oldTopicID,
				Body:             "保存した本文",
				CurrentTopicName: "Old",
			}, time.Now(), draftPageRepo, pageRepo, pageEditorRepo, topicRepo)
			if err != nil {
				t.Fatalf("saveDraftPageContent()のエラー = %v", err)
			}
			if draftPage.TopicID != pageTopicID {
				t.Errorf("下書きのTopicID = %v、期待値 = %v (ページのトピック)", draftPage.TopicID, pageTopicID)
			}
		})
	}
}

// TestSaveDraftPageContent_ConcurrentMoveは、ページを読んだ後に別のトランザクションがページを移動しても、
// 移動のコミットを待ってから移動先のトピックで下書きを保存し、移動を古いトピックで上書きしないことを確かめる
func TestSaveDraftPageContent_ConcurrentMove(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	draftPageRepo := repository.NewDraftPageRepository(q)
	pageRepo := repository.NewPageRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	topicRepo := repository.NewTopicRepository(q)

	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier("save-draft-concurrent-move").Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("save-draft-concurrent-move@example.com").
		WithAtname("savedraftconcmove").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(userID).Build()
	srcTopicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).WithNumber(1).WithName("Source").Build()
	destTopicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).WithNumber(2).WithName("Destination").Build()
	pageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(srcTopicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(srcTopicID).
		Build()

	// MovePageUsecaseと同じく、ページと下書きのトピックを移動先へ書き換え、コミットせずに保持するトランザクション
	moveTx, err := db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatalf("移動のトランザクションの開始: %v", err)
	}
	t.Cleanup(func() { _ = moveTx.Rollback() })
	var movePID int
	if err := moveTx.QueryRowContext(t.Context(), "SELECT pg_backend_pid()").Scan(&movePID); err != nil {
		t.Fatalf("移動のトランザクションの接続のPID取得: %v", err)
	}
	if _, err := pageRepo.WithTx(moveTx).MoveTopic(t.Context(), repository.MoveTopicInput{
		ID:      pageID,
		SpaceID: spaceID,
		TopicID: destTopicID,
	}); err != nil {
		t.Fatalf("ページの移動: %v", err)
	}
	if err := draftPageRepo.WithTx(moveTx).UpdateTopicByPageID(t.Context(), pageID, spaceID, destTopicID); err != nil {
		t.Fatalf("下書きのトピックの移動: %v", err)
	}

	tx, err := db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatalf("保存のトランザクションの開始: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })

	// 移動はまだコミットされていないため、保存はトランザクションの外で移動元のトピックを読んでいる
	resultCh := make(chan draftPageSaveResult, 1)
	go func() {
		draftPage, err := saveDraftPageContent(t.Context(), saveDraftPageContentInput{
			SpaceID:          spaceID,
			PageID:           pageID,
			SpaceMemberID:    spaceMemberID,
			TopicID:          srcTopicID,
			Body:             "保存した本文",
			CurrentTopicName: "Source",
		}, time.Now(), draftPageRepo.WithTx(tx), pageRepo.WithTx(tx), pageEditorRepo.WithTx(tx), topicRepo.WithTx(tx))
		resultCh <- draftPageSaveResult{draftPage: draftPage, err: err}
	}()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	waitUntilDraftPageSaveBlockedBy(t, ctx, db, movePID, "FindPageByIDForShare", resultCh)

	if err := moveTx.Commit(); err != nil {
		t.Fatalf("移動のトランザクションのコミット: %v", err)
	}

	result := <-resultCh
	if result.err != nil {
		t.Fatalf("saveDraftPageContent()のエラー = %v", result.err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("保存のトランザクションのコミット: %v", err)
	}

	draftPage, err := draftPageRepo.FindByPageAndMember(t.Context(), pageID, spaceMemberID, spaceID)
	if err != nil {
		t.Fatalf("FindByPageAndMember()のエラー = %v", err)
	}
	if draftPage == nil {
		t.Fatal("下書き = nil、期待値 = 保存した下書き")
	}
	if draftPage.TopicID != destTopicID {
		t.Errorf("下書きのTopicID = %v、期待値 = %v (移動先のトピック)", draftPage.TopicID, destTopicID)
	}
	if draftPage.Body != "保存した本文" {
		t.Errorf("下書きのBody = %q、期待値 = %q", draftPage.Body, "保存した本文")
	}
}

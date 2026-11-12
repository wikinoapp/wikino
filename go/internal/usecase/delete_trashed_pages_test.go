package usecase

import (
	"context"
	"database/sql"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func newDeleteTrashedPagesUsecase(db *sql.DB) *DeleteTrashedPagesUsecase {
	queries := query.New(db)
	return NewDeleteTrashedPagesUsecase(
		db,
		repository.NewPageRepository(queries),
		repository.NewDraftPageRepository(queries),
		repository.NewPageEditorRepository(queries),
		repository.NewPageRevisionRepository(queries),
		repository.NewPageAttachmentReferenceRepository(queries),
	)
}

// trashPageAtはページを指定した日時にゴミ箱へ入れた状態にする。
func trashPageAt(t *testing.T, db *sql.DB, spaceID model.SpaceID, pageID model.PageID, trashedAt time.Time) {
	t.Helper()

	if _, err := db.ExecContext(t.Context(), "UPDATE pages SET trashed_at = $1 WHERE space_id = $2 AND id = $3", trashedAt, spaceID, pageID); err != nil {
		t.Fatalf("ページをゴミ箱に入れるのに失敗: %v", err)
	}
}

// 定期的な物理削除はすべてのスペースを対象にするため、同じパッケージのテストどうしで消し合わない
// よう、ゴミ箱に入れた日時を他のテストより古い年にそろえる。全スペースを走査するExecuteのテストは
// このテストの中で順に確かめる。
func TestDeleteTrashedPagesUsecase(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc := newDeleteTrashedPagesUsecase(db)

	t.Run("保持期間を過ぎたページと関連レコードだけを消す", func(t *testing.T) {
		now := time.Date(1990, 3, 1, 0, 0, 0, 0, time.UTC)
		s := setupDeleteTopicSpace(t, db, "trashed-pages")
		other := setupDeleteTopicSpace(t, db, "trashed-pages-other")

		expiredTopicID, expiredPageID := seedTopicWithRelatedRecords(t, db, s, 1)
		withinTopicID, withinPageID := seedTopicWithRelatedRecords(t, db, s, 2)
		keptTopicID, keptPageID := seedTopicWithRelatedRecords(t, db, s, 3)
		otherTopicID, otherPageID := seedTopicWithRelatedRecords(t, db, other, 1)

		trashPageAt(t, db, s.spaceID, expiredPageID, now.AddDate(0, 0, -model.PageTrashRetentionDays).Add(-time.Second))
		trashPageAt(t, db, s.spaceID, withinPageID, now.AddDate(0, 0, -model.PageTrashRetentionDays).Add(time.Second))
		trashPageAt(t, db, other.spaceID, otherPageID, now.AddDate(0, 0, -60))

		if err := uc.Execute(t.Context(), DeleteTrashedPagesInput{Now: now}); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		// 保持期間を過ぎたページとその関連レコードは消え、トピックとその編集提案は残る
		expired := countTopicRelatedRows(t, db, s.spaceID, expiredTopicID, expiredPageID)
		for table, want := range map[string]int{
			"topics":                     1,
			"topic_members":              1,
			"suggestions":                1,
			"pages":                      0,
			"draft_pages":                0,
			"draft_page_revisions":       0,
			"page_editors":               0,
			"page_revisions":             0,
			"suggestion_pages":           0,
			"page_attachment_references": 0,
		} {
			if expired[table] != want {
				t.Errorf("保持期間を過ぎたページの%sの行数 = %d、期待値 = %d", table, expired[table], want)
			}
		}

		if got := countTopicRelatedRows(t, db, other.spaceID, otherTopicID, otherPageID)["pages"]; got != 0 {
			t.Errorf("別のスペースの保持期間を過ぎたページの行数 = %d、期待値 = 0", got)
		}

		// 保持期間内のページとゴミ箱に入れていないページは、関連レコードとともに残る
		for name, ids := range map[string][2]string{
			"保持期間内のページ":     {string(withinTopicID), string(withinPageID)},
			"ゴミ箱に入れていないページ": {string(keptTopicID), string(keptPageID)},
		} {
			rows := countTopicRelatedRows(t, db, s.spaceID, model.TopicID(ids[0]), model.PageID(ids[1]))
			for table, n := range rows {
				if n != 1 {
					t.Errorf("%sの%sの行数 = %d、期待値 = 1", name, table, n)
				}
			}
		}

		// 添付ファイル自体はスペースに属するため消さない
		if got := countRows(t, db, "attachments", s.spaceID, "id = $2", s.attachmentID); got != 1 {
			t.Errorf("添付ファイルの行数 = %d、期待値 = 1", got)
		}
	})

	t.Run("上限の件数ずつ消し、残りを次の繰り返しで消す", func(t *testing.T) {
		now := time.Date(1980, 3, 1, 0, 0, 0, 0, time.UTC)
		trashedAt := now.AddDate(0, 0, -60)
		s := setupDeleteTopicSpace(t, db, "trashed-pages-batch")
		topicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(s.spaceID).WithNumber(1).WithName("上限").Build()
		for i := range 5 {
			testutil.NewPageBuilderDB(t, db).
				WithSpaceID(s.spaceID).
				WithTopicID(topicID).
				WithNumber(model.PageNumber(i + 1)).
				WithTitle(string(rune('A' + i))).
				WithTrashedAt(trashedAt.Add(time.Duration(i) * time.Minute)).
				Build()
		}
		countPages := func() int { return countRows(t, db, "pages", s.spaceID, "topic_id = $2", topicID) }
		trashedBefore := now.AddDate(0, 0, -model.PageTrashRetentionDays)

		deleted, err := uc.deleteBatch(t.Context(), s.spaceID, trashedBefore, 2)
		if err != nil {
			t.Fatalf("deleteBatch()のエラー = %v", err)
		}
		if deleted != 2 || countPages() != 3 {
			t.Errorf("1回目に消した件数 = %d、残り = %d、期待値 = 2件消して3件残る", deleted, countPages())
		}

		if err := uc.Execute(t.Context(), DeleteTrashedPagesInput{Now: now, BatchSize: 2}); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if got := countPages(); got != 0 {
			t.Errorf("残ったページ = %d、期待値 = 0", got)
		}
	})
}

// IDとゴミ箱へ入れた日時の順が逆でも、トピック削除と定期削除のロックが循環待ちしないことを確認する。
func TestDeleteTrashedPagesUsecase_トピック削除と同じ順序でロックする(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	db := testutil.GetTestDB()
	s := setupDeleteTopicSpace(t, db, "trashed-pages-lock-order")
	topicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(s.spaceID).WithNumber(1).WithName("ロックの順序").Build()
	pageIDs := []model.PageID{
		testutil.NewPageBuilderDB(t, db).WithSpaceID(s.spaceID).WithTopicID(topicID).WithNumber(1).WithTitle("A").Build(),
		testutil.NewPageBuilderDB(t, db).WithSpaceID(s.spaceID).WithTopicID(topicID).WithNumber(2).WithTitle("B").Build(),
	}
	slices.Sort(pageIDs)
	// 全スペースを走査するテストの起点より新しくし、並行実行してもこのテストのページを消されないようにする。
	now := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
	trashPageAt(t, db, s.spaceID, pageIDs[0], now.AddDate(0, 0, -31))
	trashPageAt(t, db, s.spaceID, pageIDs[1], now.AddDate(0, 0, -32))
	discardTopicForTest(t, db, s.spaceID, topicID)

	topicTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("トピック削除のトランザクションの開始に失敗: %v", err)
	}
	defer func() { _ = topicTx.Rollback() }()
	var blockingPID int
	if err := topicTx.QueryRowContext(ctx, "SELECT pg_backend_pid()").Scan(&blockingPID); err != nil {
		t.Fatalf("トピック削除の接続の取得に失敗: %v", err)
	}
	if _, err := repository.NewTopicRepository(query.New(topicTx)).LockDiscarded(ctx, topicID, s.spaceID); err != nil {
		t.Fatalf("トピックのロックに失敗: %v", err)
	}
	// トピック削除が先頭のページだけをロックした時点を固定する。
	var lockedID string
	if err := topicTx.QueryRowContext(ctx, "SELECT id FROM pages WHERE space_id = $1 AND id = $2 FOR UPDATE", s.spaceID, pageIDs[0]).Scan(&lockedID); err != nil {
		t.Fatalf("先頭のページのロックに失敗: %v", err)
	}
	result := make(chan error, 1)
	go func() {
		_, err := newDeleteTrashedPagesUsecase(db).deleteBatch(ctx, s.spaceID, now.AddDate(0, 0, -model.PageTrashRetentionDays), 100)
		result <- err
	}()
	waitForQueryBlockedBy(t, ctx, db, blockingPID, "ListTrashedPageIDsBeforeForUpdate", result)

	ids, err := repository.NewPageRepository(query.New(topicTx)).ListIDsByTopicForUpdate(ctx, topicID, s.spaceID)
	if err != nil {
		t.Fatalf("トピック削除のページのロックに失敗: %v", err)
	}
	if !slices.Equal(ids, pageIDs) {
		t.Errorf("トピック削除のページ = %v、期待値 = %v", ids, pageIDs)
	}
	if err := topicTx.Rollback(); err != nil {
		t.Fatalf("トピック削除のロック解放に失敗: %v", err)
	}
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("定期削除のエラー = %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("定期削除が完了しなかった: %v", ctx.Err())
	}
	if got := countRows(t, db, "pages", s.spaceID, "topic_id = $2", topicID); got != 0 {
		t.Errorf("定期削除後に残ったページ = %d、期待値 = 0", got)
	}
}

// 復元のコミットを待ってから期限を確かめ、復元されたページと関連レコードを消さないことを確認する。
func TestDeleteTrashedPagesUsecase_ロック待ち中に復元されたページを保持する(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	db := testutil.GetTestDB()
	s := setupDeleteTopicSpace(t, db, "trashed-pages-restored")
	topicID, pageID := seedTopicWithRelatedRecords(t, db, s, 1)
	now := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
	trashPageAt(t, db, s.spaceID, pageID, now.AddDate(0, 0, -31))
	before := countTopicRelatedRows(t, db, s.spaceID, topicID, pageID)

	restoreTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("復元のトランザクションの開始に失敗: %v", err)
	}
	defer func() { _ = restoreTx.Rollback() }()
	var blockingPID int
	if err := restoreTx.QueryRowContext(ctx, "SELECT pg_backend_pid()").Scan(&blockingPID); err != nil {
		t.Fatalf("復元の接続の取得に失敗: %v", err)
	}
	if _, err := repository.NewPageRepository(query.New(restoreTx)).RestoreFromTrash(ctx, []model.PageID{pageID}, []model.TopicID{topicID}, s.spaceID, now); err != nil {
		t.Fatalf("ページの復元に失敗: %v", err)
	}
	result := make(chan error, 1)
	go func() {
		_, err := newDeleteTrashedPagesUsecase(db).deleteBatch(ctx, s.spaceID, now.AddDate(0, 0, -model.PageTrashRetentionDays), 100)
		result <- err
	}()
	waitForQueryBlockedBy(t, ctx, db, blockingPID, "ListTrashedPageIDsBeforeForUpdate", result)
	if err := restoreTx.Commit(); err != nil {
		t.Fatalf("復元のコミットに失敗: %v", err)
	}
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("定期削除のエラー = %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("定期削除が完了しなかった: %v", ctx.Err())
	}
	if after := countTopicRelatedRows(t, db, s.spaceID, topicID, pageID); !reflect.DeepEqual(after, before) {
		t.Errorf("復元されたページと関連レコード = %v、期待値 = %v", after, before)
	}
	if isPageTrashed(t, db, pageID) {
		t.Error("復元されたページがゴミ箱に残っている")
	}
}

// 関連レコードの一部を消した後に失敗しても、ページと関連レコードがすべて残ることを確認する。
func TestDeleteTrashedPagesUsecase_削除途中の失敗を取り消す(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	db := testutil.GetTestDB()
	s := setupDeleteTopicSpace(t, db, "trashed-pages-rollback")
	topicID, pageID := seedTopicWithRelatedRecords(t, db, s, 1)
	now := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
	trashPageAt(t, db, s.spaceID, pageID, now.AddDate(0, 0, -31))
	before := countTopicRelatedRows(t, db, s.spaceID, topicID, pageID)

	blocker, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("添付参照をロックするトランザクションの開始に失敗: %v", err)
	}
	defer func() { _ = blocker.Rollback() }()
	var blockingPID int
	if err := blocker.QueryRowContext(ctx, "SELECT pg_backend_pid()").Scan(&blockingPID); err != nil {
		t.Fatalf("添付参照をロックする接続の取得に失敗: %v", err)
	}
	var lockedID string
	if err := blocker.QueryRowContext(ctx, `SELECT par.id FROM page_attachment_references par
		INNER JOIN pages p ON p.id = par.page_id
		WHERE p.space_id = $1 AND p.id = $2 FOR UPDATE OF par`, s.spaceID, pageID).Scan(&lockedID); err != nil {
		t.Fatalf("添付参照のロックに失敗: %v", err)
	}

	deleteCtx, cancelDelete := context.WithCancel(ctx)
	defer cancelDelete()
	result := make(chan error, 1)
	go func() {
		_, err := newDeleteTrashedPagesUsecase(db).deleteBatch(deleteCtx, s.spaceID, now.AddDate(0, 0, -model.PageTrashRetentionDays), 100)
		result <- err
	}()
	// 下書き・編集者・リビジョンの削除が済み、最後の添付参照の削除を待っている状態で失敗させる。
	waitForQueryBlockedBy(t, ctx, db, blockingPID, "DeletePageAttachmentReferencesByPageIDs", result)
	cancelDelete()
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("キャンセルしたのに定期削除が成功した")
		}
	case <-ctx.Done():
		t.Fatalf("定期削除の取り消しが完了しなかった: %v", ctx.Err())
	}
	if err := blocker.Rollback(); err != nil {
		t.Fatalf("添付参照のロック解放に失敗: %v", err)
	}
	if after := countTopicRelatedRows(t, db, s.spaceID, topicID, pageID); !reflect.DeepEqual(after, before) {
		t.Errorf("取り消したページと関連レコード = %v、期待値 = %v", after, before)
	}
	if !isPageTrashed(t, db, pageID) {
		t.Error("削除を取り消したページがゴミ箱から外れた")
	}
}

package usecase

import (
	"context"
	"database/sql"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// 共通のリンク元をロックし、両方の公開が自身のページを更新してから付け替えへ進むようにする。
// ロックを解放すると付け替えとページの更新の間でデッドロックが起こるが、公開全体の再試行で回復する
func TestPagePublication_ConcurrentReplacements(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		kind string
		name string
	}{
		{kind: "web", name: "Web画面の公開どうし"},
		{kind: "suggestion", name: "Web画面の公開と編集提案の反映"},
		{kind: "api-update", name: "Web画面の公開とAPIの更新"},
		{kind: "api-draft-publication", name: "Web画面の公開とAPIの下書きの公開"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			kind := tt.kind
			db := testutil.GetTestDB()
			f := setupCreateAPIPageFixture(t, db, "publish-deadlock-"+kind+"-"+uuid.NewString()[:8], model.SpaceRoleEditor)
			spaceID, topicID := f.space.ID, f.topicIDs[1]
			testutil.NewTopicMemberBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).
				WithSpaceMemberID(f.spaceMemberID).WithRole(model.TopicRoleEditor).Build()
			oldA := testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(1).
				WithTitle("A").WithBody("").WithUnpublished().Build()
			oldB := testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(2).
				WithTitle("B").WithBody("").WithUnpublished().Build()
			// 更新の対象になる行のうち、この行を先に作って付け替えの先頭で待たせる
			sourceID := testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(3).
				WithTitle("リンク元").WithBody("[[A]] [[B]]").WithLinkedPageIDs([]model.PageID{oldA, oldB}).Build()
			pageA := testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(4).
				WithTitle("元のA").WithBody("[[B]]").WithLinkedPageIDs([]model.PageID{oldB}).Build()
			pageB := testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(5).
				WithTitle("元のB").WithBody("[[A]]").WithLinkedPageIDs([]model.PageID{oldA}).Build()
			sourceBefore := findPageForTest(t, db, spaceID, 3)
			publish := newConcurrentPublishPageUC(db)
			principal := f.principal(model.SpaceRoleEditor, draftPagePublishScopes)
			second := func(ctx context.Context) (model.PageID, error) {
				output, err := publish.Execute(ctx, PublishPageInput{
					SpaceIdentifier: f.space.Identifier, PageNumber: 5, UserID: f.userID, Title: "B", Body: "[[A]]",
				})
				if err != nil {
					return "", err
				}
				return output.Page.ID, nil
			}
			switch kind {
			case "suggestion":
				suggestionID := testutil.NewSuggestionBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).
					WithCreatedSpaceMemberID(f.spaceMemberID).WithStatus(model.SuggestionStatusOpen).Build()
				testutil.NewSuggestionPageBuilderDB(t, db).WithSpaceID(spaceID).WithSuggestionID(suggestionID).WithPageID(pageB).
					WithTitle("B").WithBody("[[A]]").WithLinkedPageIDs([]model.PageID{oldA}).Build()
				uc := newConcurrentApplySuggestionUC(db)
				second = func(ctx context.Context) (model.PageID, error) {
					_, err := uc.Execute(ctx, ApplySuggestionInput{SpaceIdentifier: f.space.Identifier, SuggestionNumber: 1, UserID: f.userID})
					return pageB, err
				}
			case "api-update":
				second = func(ctx context.Context) (model.PageID, error) {
					output, err := newUpdateAPIPageUC(db).Execute(ctx, UpdateAPIPageInput{
						Principal: principal, SpaceIdentifier: f.space.Identifier, PageNumber: 5, IfMatchAny: true, Title: strPtr("B"),
					})
					if err != nil {
						return "", err
					}
					return output.Page.ID, nil
				}
			case "api-draft-publication":
				testutil.NewDraftPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithSpaceMemberID(f.spaceMemberID).
					WithPageID(pageB).WithTitle("B").WithBody("[[A]]").WithLinkedPageIDs([]model.PageID{oldA}).Build()
				second = func(ctx context.Context) (model.PageID, error) {
					output, err := newCreateAPIDraftPagePublicationUC(db).Execute(ctx, CreateAPIDraftPagePublicationInput{
						Principal: principal, SpaceIdentifier: f.space.Identifier, PageNumber: 5, IfMatchAny: true,
					})
					if err != nil {
						return "", err
					}
					return output.Page.ID, nil
				}
			}

			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			blocker, blockerPID := beginWebTx(t, db)
			if _, err := blocker.ExecContext(ctx, "SELECT id FROM pages WHERE id = $1 AND space_id = $2 FOR NO KEY UPDATE", sourceID, spaceID); err != nil {
				t.Fatalf("リンク元のロック: %v", err)
			}
			results := make(chan concurrentPublicationResult, 2)
			go func() {
				output, err := publish.Execute(ctx, PublishPageInput{
					SpaceIdentifier: f.space.Identifier, PageNumber: 4, UserID: f.userID, Title: "A", Body: "[[B]]",
				})
				id := pageA
				if output != nil {
					id = output.Page.ID
				}
				results <- concurrentPublicationResult{id, err}
			}()
			go func() {
				id, err := second(ctx)
				results <- concurrentPublicationResult{id, err}
			}()
			waitForPublicationsBlocked(t, ctx, db, blockerPID, results)
			if err := blocker.Rollback(); err != nil {
				t.Fatalf("リンク元のロック解除: %v", err)
			}
			for range 2 {
				result := <-results
				if result.err != nil {
					t.Fatalf("公開のエラー = %v、期待値 = nil", result.err)
				}
				if result.pageID != pageA {
					pageB = result.pageID
				}
			}
			q := query.New(db)
			pages, err := repository.NewPageRepository(q).FindByIDs(ctx, []model.PageID{pageA, pageB, sourceID}, spaceID)
			if err != nil {
				t.Fatalf("公開後のページの取得: %v", err)
			}
			byID := make(map[model.PageID]*model.Page, len(pages))
			for _, p := range pages {
				byID[p.ID] = p
			}
			for _, id := range []model.PageID{pageA, pageB} {
				p := byID[id]
				if p == nil || !p.IsPublished() {
					t.Fatalf("公開したページが見つかりません: %v", id)
				}
				wantTitle, wantLinks := "A", []model.PageID{pageB}
				if id == pageB {
					wantTitle, wantLinks = "B", []model.PageID{pageA}
				}
				if derefOr(p.Title, "") != wantTitle || !slices.Equal(p.LinkedPageIDs, wantLinks) {
					t.Errorf("ページのタイトルとリンク = %v / %v、期待値 = %v / %v", p.Title, p.LinkedPageIDs, wantTitle, wantLinks)
				}
				if n := countRows(t, db, "page_revisions", spaceID, "page_id = $2", id); n != 1 {
					t.Errorf("公開の編集履歴の数 = %d、期待値 = 1", n)
				}
			}
			source := byID[sourceID]
			if source == nil || !slices.Equal(source.LinkedPageIDs, []model.PageID{pageA, pageB}) || source.ContentDigest() != sourceBefore.ContentDigest() {
				t.Errorf("リンク元 = %+v、期待値 = 両方の公開先へ付け替わり、本文と版が変わらない", source)
			}
		})
	}
}

type concurrentPublicationResult struct {
	pageID model.PageID
	err    error
}

// リンクだけを読み直す際に、取得後に本文が変わった編集提案を未検証のまま反映しない
func TestApplySuggestion_ContentChangedBeforeLock(t *testing.T) {
	t.Parallel()
	db := testutil.GetTestDB()
	f := setupCreateAPIPageFixture(t, db, "apply-content-before-lock", model.SpaceRoleEditor)
	pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
		WithTitle("ページ").WithBody("公開済みの本文").Build()
	suggestionID := testutil.NewSuggestionBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).
		WithCreatedSpaceMemberID(f.spaceMemberID).WithStatus(model.SuggestionStatusOpen).Build()
	spID := testutil.NewSuggestionPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithSuggestionID(suggestionID).WithPageID(pageID).
		WithTitle("ページ").WithBody("検証した本文").Build()
	uc := newConcurrentApplySuggestionUC(db)
	data, err := uc.fetchData(t.Context(), ApplySuggestionInput{SpaceIdentifier: f.space.Identifier, SuggestionNumber: 1, UserID: f.userID})
	if err != nil {
		t.Fatal(err)
	}
	execForTest(t, db, "UPDATE suggestion_pages SET body = '変更された本文', updated_at = updated_at + interval '1 second' WHERE id = $1 AND space_id = $2", spID, f.space.ID)
	_, err = uc.applySuggestion(t.Context(), data, nil)
	if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeConflict {
		t.Fatalf("反映のエラー = %v、期待値 = AppErrCodeConflict", err)
	}
	if body := findPageForTest(t, db, f.space.ID, 1).Body; body != "公開済みの本文" {
		t.Errorf("本文 = %q、期待値 = 公開済みの本文", body)
	}
}

func newConcurrentApplySuggestionUC(db *sql.DB) *ApplySuggestionUsecase {
	q := query.New(db)
	return NewApplySuggestionUsecase(db, repository.NewSpaceRepository(q), repository.NewSpaceMemberRepository(q),
		repository.NewTopicMemberRepository(q), repository.NewSuggestionRepository(q), repository.NewSuggestionPageRepository(q),
		repository.NewPageRepository(q), repository.NewPageRevisionRepository(q), repository.NewPageEditorRepository(q),
		repository.NewAttachmentRepository(q), repository.NewPageAttachmentReferenceRepository(q), repository.NewDraftPageRepository(q),
		validator.NewSuggestionApplyValidator(validator.NewPageUpdateValidator(repository.NewPageRepository(q))))
}

func newConcurrentPublishPageUC(db *sql.DB) *PublishPageUsecase {
	q := query.New(db)
	pageRepo := repository.NewPageRepository(q)
	return NewPublishPageUsecase(db, repository.NewSpaceRepository(q), repository.NewSpaceMemberRepository(q), pageRepo,
		repository.NewPageRevisionRepository(q), repository.NewPageEditorRepository(q), repository.NewDraftPageRepository(q),
		repository.NewDraftPageRevisionRepository(q), repository.NewSuggestionPageRepository(q), repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q), repository.NewAttachmentRepository(q), repository.NewPageAttachmentReferenceRepository(q),
		validator.NewPageUpdateValidator(pageRepo))
}

// 付け替えのUPDATEが共通のリンク元のロックに並ぶまで待つ。
// 後続は先行の行ロック待ちに並ぶこともあるため、待機している接続を再帰的に辿る
func waitForPublicationsBlocked(t *testing.T, ctx context.Context, db *sql.DB, blockerPID int, results <-chan concurrentPublicationResult) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var n int
		err := db.QueryRowContext(ctx, `WITH RECURSIVE blocked(pid) AS (
            SELECT pid FROM pg_stat_activity WHERE datname = current_database() AND $1 = ANY(pg_blocking_pids(pid))
            UNION
            SELECT a.pid FROM pg_stat_activity a JOIN blocked b ON b.pid = ANY(pg_blocking_pids(a.pid)) WHERE a.datname = current_database()
        ) SELECT count(*) FROM pg_stat_activity JOIN blocked USING (pid) WHERE query LIKE '%ReassignPageLinkedPageID%'`, blockerPID).Scan(&n)
		if err != nil {
			t.Fatalf("公開のロック待ちの確認: %v", err)
		}
		if n == 2 {
			return
		}
		select {
		case result := <-results:
			t.Fatalf("競合前に公開が終了: %v", result.err)
		case <-ctx.Done():
			t.Fatalf("両方の公開が付け替えのロックを待ちませんでした: %v", ctx.Err())
		case <-ticker.C:
		}
	}
}

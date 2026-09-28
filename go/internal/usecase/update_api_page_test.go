package usecase

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func newUpdateAPIPageUC(db *sql.DB) *UpdateAPIPageUsecase {
	q := query.New(db)
	pageRepo := repository.NewPageRepository(q)
	return NewUpdateAPIPageUsecase(
		db,
		repository.NewSpaceRepository(q),
		pageRepo,
		repository.NewPageRevisionRepository(q),
		repository.NewPageEditorRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
		repository.NewAttachmentRepository(q),
		repository.NewPageAttachmentReferenceRepository(q),
		validator.NewAPIPageUpdateValidator(pageRepo),
	)
}

// findPageForTestはページ番号でページを取得する。存在しなければテストを失敗させる
func findPageForTest(t *testing.T, db *sql.DB, spaceID model.SpaceID, number model.PageNumber) *model.Page {
	t.Helper()

	page, err := repository.NewPageRepository(query.New(db)).FindBySpaceAndNumber(t.Context(), spaceID, number)
	if err != nil || page == nil {
		t.Fatalf("ページ %d の取得: page=%v, err=%v", number, page, err)
	}
	return page
}

func TestUpdateAPIPageUsecase_Execute(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberScopes := apiTopicRegularMemberScopes
	tokenScopes := []model.Scope{model.ScopePageWrite}

	t.Run("タイトルと本文を更新し、下書きと公開日時に触れない", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAPIPageFixture(t, db, "api-page-update", memberScopes)
		publishedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
			WithTitle("元のタイトル").WithBody("元の本文").WithPublishedAt(publishedAt).Build()
		testutil.NewDraftPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithPageID(pageID).WithTopicID(f.topicIDs[1]).
			WithSpaceMemberID(f.spaceMemberID).WithTitle("下書きのタイトル").WithBody("下書きの本文").Build()
		before := findPageForTest(t, db, f.space.ID, 1)

		// 本文は新しいタイトルで自身を指すWikiリンクと、存在しないページへのWikiリンクを含む
		body := "[[新しいタイトル]] [[まだ無いページ]]"
		output, err := newUpdateAPIPageUC(db).Execute(t.Context(), UpdateAPIPageInput{
			Principal:       f.principal(memberScopes, tokenScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			IfMatchDigests:  []string{"古い版", before.ContentDigest()},
			Title:           strPtr("新しいタイトル"),
			Body:            strPtr(body),
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		page := output.Page
		if page.ID != pageID || page.Title == nil || *page.Title != "新しいタイトル" || page.Body != body {
			t.Errorf("ページ = %+v、期待値 = 更新したページ", page)
		}
		if !page.ModifiedAt.After(before.ModifiedAt) {
			t.Errorf("ModifiedAt = %s、期待値 = %s より後", page.ModifiedAt, before.ModifiedAt)
		}
		if page.PublishedAt == nil || !page.PublishedAt.Equal(publishedAt) {
			t.Errorf("PublishedAt = %v、期待値 = %s のまま", page.PublishedAt, publishedAt)
		}
		if page.ContentDigest() == before.ContentDigest() {
			t.Error("更新後の版が更新前と同じ")
		}
		if output.Topic.ID != f.topicIDs[1] {
			t.Errorf("Topic.ID = %s、期待値 = %s", output.Topic.ID, f.topicIDs[1])
		}

		// 自身へのリンクで同じタイトルのページを別に作らず、存在しないページだけを下書きで作る
		if n := countRows(t, db, "pages", f.space.ID, "title = $2", "新しいタイトル"); n != 1 {
			t.Errorf("「新しいタイトル」の数 = %d、期待値 = 1", n)
		}
		if n := countRows(t, db, "pages", f.space.ID, "title = $2 AND published_at IS NULL", "まだ無いページ"); n != 1 {
			t.Errorf("リンク先として作成された「まだ無いページ」の数 = %d、期待値 = 1", n)
		}
		if len(page.LinkedPageIDs) != 2 || page.LinkedPageIDs[0] != pageID {
			t.Errorf("LinkedPageIDs = %v、期待値 = 自分と作成したページ", page.LinkedPageIDs)
		}

		if n := countRows(t, db, "page_revisions", f.space.ID, "page_id = $2 AND space_member_id = $3 AND title = $4", pageID, f.spaceMemberID, "新しいタイトル"); n != 1 {
			t.Errorf("リビジョンの数 = %d、期待値 = 1", n)
		}
		if n := countRows(t, db, "page_editors", f.space.ID, "page_id = $2 AND space_member_id = $3", pageID, f.spaceMemberID); n != 1 {
			t.Errorf("編集者の数 = %d、期待値 = 1", n)
		}
		if n := countRows(t, db, "draft_pages", f.space.ID, "page_id = $2 AND title = $3 AND body = $4", pageID, "下書きのタイトル", "下書きの本文"); n != 1 {
			t.Errorf("下書きの数 = %d、期待値 = 1 (更新前のまま残る)", n)
		}
	})

	t.Run("含めなかった項目は変えない", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAPIPageFixture(t, db, "api-page-update-partial", memberScopes)
		testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
			WithTitle("タイトル").WithBody("元の本文").Build()
		before := findPageForTest(t, db, f.space.ID, 1)

		output, err := newUpdateAPIPageUC(db).Execute(t.Context(), UpdateAPIPageInput{
			Principal:       f.principal(memberScopes, tokenScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			IfMatchDigests:  []string{before.ContentDigest()},
			Body:            strPtr("新しい本文"),
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.Page.Title == nil || *output.Page.Title != "タイトル" || output.Page.Body != "新しい本文" {
			t.Errorf("ページ = %+v、期待値 = タイトルはそのままで本文だけ更新", output.Page)
		}
	})

	t.Run("タイトルと本文が今と同じなら何も書き込まない", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAPIPageFixture(t, db, "api-page-update-noop", memberScopes)
		testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
			WithTitle("タイトル").WithBody("本文").Build()
		before := findPageForTest(t, db, f.space.ID, 1)

		output, err := newUpdateAPIPageUC(db).Execute(t.Context(), UpdateAPIPageInput{
			Principal:       f.principal(memberScopes, tokenScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			IfMatchDigests:  []string{before.ContentDigest()},
			Title:           strPtr("タイトル"),
			Body:            strPtr("本文"),
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.Page.ContentDigest() != before.ContentDigest() {
			t.Error("何も変えない更新で版が変わった")
		}
		if n := countRows(t, db, "page_revisions", f.space.ID, ""); n != 0 {
			t.Errorf("リビジョンの数 = %d、期待値 = 0", n)
		}
	})

	t.Run("タイトルを変えなければ、今の規則に合わないタイトルのページも本文を更新できる", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAPIPageFixture(t, db, "api-page-update-legacy", memberScopes)
		testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
			WithTitle("古い:タイトル").WithBody("本文").Build()
		before := findPageForTest(t, db, f.space.ID, 1)

		if _, err := newUpdateAPIPageUC(db).Execute(t.Context(), UpdateAPIPageInput{
			Principal:       f.principal(memberScopes, tokenScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			IfMatchDigests:  []string{before.ContentDigest()},
			Title:           strPtr("古い:タイトル"),
			Body:            strPtr("新しい本文"),
		}); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	})
}

func TestUpdateAPIPageUsecase_Execute_Precondition(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberScopes := apiTopicRegularMemberScopes
	tokenScopes := []model.Scope{model.ScopePageWrite}

	tests := []struct {
		name       string
		digests    func(current string) []string
		any        bool
		wantFailed bool
	}{
		{name: "今の版と一致すれば更新する", digests: func(current string) []string { return []string{current} }},
		{name: "`*` なら版を問わず更新する", any: true},
		{name: "今の版と一致しなければ412", digests: func(string) []string { return []string{"古い版"} }, wantFailed: true},
		{name: "候補が無ければ412", digests: func(string) []string { return nil }, wantFailed: true},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupCreateAPIPageFixture(t, db, "api-page-update-pre-"+string(rune('a'+i)), memberScopes)
			testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
				WithTitle("タイトル").WithBody("本文").Build()
			current := findPageForTest(t, db, f.space.ID, 1).ContentDigest()
			var digests []string
			if tt.digests != nil {
				digests = tt.digests(current)
			}

			output, err := newUpdateAPIPageUC(db).Execute(t.Context(), UpdateAPIPageInput{
				Principal:       f.principal(memberScopes, tokenScopes),
				SpaceIdentifier: f.space.Identifier,
				PageNumber:      1,
				IfMatchDigests:  digests,
				IfMatchAny:      tt.any,
				Body:            strPtr("新しい本文"),
			})

			if !tt.wantFailed {
				if err != nil {
					t.Fatalf("Execute() error = %v", err)
				}
				return
			}
			if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodePreconditionFailed {
				t.Errorf("error = %v、期待値 = AppErrCodePreconditionFailed", err)
			}
			if output != nil {
				t.Errorf("output = %+v、期待値 = nil", output)
			}
			if got := findPageForTest(t, db, f.space.ID, 1).Body; got != "本文" {
				t.Errorf("本文 = %q、期待値 = 更新されない", got)
			}
		})
	}
}

func TestUpdateAPIPageUsecase_Execute_Authorization(t *testing.T) {
	t.Parallel()

	writeScopes := []model.Scope{model.ScopePageWrite}
	writeAndTopicScopes := []model.Scope{model.ScopePageWrite, model.ScopeTopicRead}
	tests := []struct {
		name         string
		memberScopes []model.Scope
		tokenScopes  []model.Scope
		identifier   model.SpaceIdentifier
		topicNumber  int32
		// setupはページの状態を変える
		setup    func(t *testing.T, db *sql.DB, spaceID model.SpaceID, pageID model.PageID)
		wantCode model.AppErrorCode
	}{
		{
			name:         "トークンがtopic:readを持てば参加している非公開トピックのページを更新できる",
			memberScopes: apiTopicRegularMemberScopes,
			tokenScopes:  writeAndTopicScopes,
			topicNumber:  2,
		},
		{
			name:         "トークンがtopic:readを持たなければ参加している非公開トピックのページは未存在",
			memberScopes: apiTopicRegularMemberScopes,
			tokenScopes:  writeScopes,
			topicNumber:  2,
			wantCode:     model.AppErrCodeResourceNotFound,
		},
		{
			name:         "参加していない非公開トピックのページは未存在",
			memberScopes: apiTopicRegularMemberScopes,
			tokenScopes:  writeAndTopicScopes,
			topicNumber:  3,
			wantCode:     model.AppErrCodeResourceNotFound,
		},
		{
			name:         "メンバーがpage:writeを持たなければ未存在",
			memberScopes: []model.Scope{model.ScopePageRead, model.ScopePersonalAccessTokenWrite},
			tokenScopes:  writeScopes,
			topicNumber:  1,
			wantCode:     model.AppErrCodeForbidden,
		},
		{
			name:         "トークンがpage:writeを持たなければ、メンバーが持っていても更新できない",
			memberScopes: apiTopicAdminMemberScopes,
			tokenScopes:  []model.Scope{model.ScopePageRead, model.ScopeTopicRead},
			topicNumber:  1,
			wantCode:     model.AppErrCodeForbidden,
		},
		{
			name:         "束縛先と異なるスペースは未存在",
			memberScopes: apiTopicRegularMemberScopes,
			tokenScopes:  writeScopes,
			identifier:   "api-page-update-auth-other",
			topicNumber:  1,
			wantCode:     model.AppErrCodeResourceNotFound,
		},
		{
			name:         "未公開のページは未存在",
			memberScopes: apiTopicRegularMemberScopes,
			tokenScopes:  writeScopes,
			topicNumber:  1,
			setup: func(t *testing.T, db *sql.DB, spaceID model.SpaceID, pageID model.PageID) {
				execForTest(t, db, "UPDATE pages SET published_at = NULL WHERE id = $1 AND space_id = $2", pageID, spaceID)
			},
			wantCode: model.AppErrCodeResourceNotFound,
		},
		{
			name:         "ゴミ箱のページは未存在",
			memberScopes: apiTopicAdminMemberScopes,
			tokenScopes:  writeAndTopicScopes,
			topicNumber:  1,
			setup: func(t *testing.T, db *sql.DB, spaceID model.SpaceID, pageID model.PageID) {
				execForTest(t, db, "UPDATE pages SET trashed_at = now() WHERE id = $1 AND space_id = $2", pageID, spaceID)
			},
			wantCode: model.AppErrCodeResourceNotFound,
		},
	}

	db := testutil.GetTestDB()
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupCreateAPIPageFixture(t, db, "api-page-update-auth-"+string(rune('a'+i)), tt.memberScopes)
			pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[tt.topicNumber]).WithNumber(1).
				WithTitle("タイトル").WithBody("本文").Build()
			if tt.setup != nil {
				tt.setup(t, db, f.space.ID, pageID)
			}
			identifier := f.space.Identifier
			if tt.identifier != "" {
				identifier = tt.identifier
			}

			output, err := newUpdateAPIPageUC(db).Execute(t.Context(), UpdateAPIPageInput{
				Principal:       f.principal(tt.memberScopes, tt.tokenScopes),
				SpaceIdentifier: identifier,
				PageNumber:      1,
				IfMatchAny:      true,
				Body:            strPtr("新しい本文"),
			})

			if tt.wantCode == 0 {
				if err != nil {
					t.Fatalf("Execute() error = %v", err)
				}
				if output.Page.Body != "新しい本文" {
					t.Errorf("本文 = %q、期待値 = 新しい本文", output.Page.Body)
				}
				return
			}

			if ae := model.AsAppError(err); ae == nil || ae.Code != tt.wantCode {
				t.Errorf("error = %v、期待値 = AppErrorCode %d", err, tt.wantCode)
			}
			if n := countRows(t, db, "pages", f.space.ID, "id = $2 AND body = $3", pageID, "本文"); n != 1 {
				t.Error("ページが更新された")
			}
		})
	}
}

func TestUpdateAPIPageUsecase_Execute_TitleConflict(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberScopes := apiTopicRegularMemberScopes
	tokenScopes := []model.Scope{model.ScopePageWrite}

	t.Run("公開済みの他のページと同じタイトルは422", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAPIPageFixture(t, db, "api-page-update-dup", memberScopes)
		testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("更新するページ").Build()
		testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(2).WithTitle("重複").Build()

		output, err := newUpdateAPIPageUC(db).Execute(t.Context(), UpdateAPIPageInput{
			Principal:       f.principal(memberScopes, tokenScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			IfMatchAny:      true,
			Title:           strPtr("重複"),
		})
		if output != nil {
			t.Errorf("output = %+v、期待値 = nil", output)
		}
		if ve := model.AsValidationError(err); ve == nil || !ve.HasFieldError("title") {
			t.Errorf("error = %v、期待値 = titleのValidationError", err)
		}
	})

	t.Run("Wikiリンクで作られた中身の無い未公開のページは論理削除して置き換える", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAPIPageFixture(t, db, "api-page-update-linked", memberScopes)
		testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("更新するページ").Build()
		linkedID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(2).
			WithTitle("リンク先").WithBody("").WithUnpublished().Build()

		output, err := newUpdateAPIPageUC(db).Execute(t.Context(), UpdateAPIPageInput{
			Principal:       f.principal(memberScopes, tokenScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			IfMatchAny:      true,
			Title:           strPtr("リンク先"),
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.Page.Number != 1 || output.Page.Title == nil || *output.Page.Title != "リンク先" {
			t.Errorf("ページ = %+v、期待値 = タイトルを「リンク先」にしたページ1", output.Page)
		}
		if n := countRows(t, db, "pages", f.space.ID, "id = $2 AND discarded_at IS NOT NULL", linkedID); n != 1 {
			t.Error("未公開のページが論理削除されていない")
		}
	})
}

// TestUpdateAPIPageUsecase_ExecuteConcurrentlyは、同じ版を前提にした更新を同時に送ると、
// 1件だけが更新され、残りは前提の不一致になることを確かめる
func TestUpdateAPIPageUsecase_ExecuteConcurrently(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberScopes := apiTopicRegularMemberScopes
	f := setupCreateAPIPageFixture(t, db, "api-page-update-cc-"+uuid.NewString()[:8], memberScopes)
	testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
		WithTitle("タイトル").WithBody("本文").Build()
	digest := findPageForTest(t, db, f.space.ID, 1).ContentDigest()
	uc := newUpdateAPIPageUC(db)

	const concurrency = 4
	errs := make([]error, concurrency)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, errs[i] = uc.Execute(t.Context(), UpdateAPIPageInput{
				Principal:       f.principal(memberScopes, []model.Scope{model.ScopePageWrite}),
				SpaceIdentifier: f.space.Identifier,
				PageNumber:      1,
				IfMatchDigests:  []string{digest},
				Body:            strPtr("並行した更新" + string(rune('A'+i))),
			})
		}()
	}
	close(start)
	wg.Wait()

	succeeded := 0
	for i, err := range errs {
		if err == nil {
			succeeded++
			continue
		}
		if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodePreconditionFailed {
			t.Errorf("更新%dのエラー = %v、期待値 = AppErrCodePreconditionFailed", i, err)
		}
	}
	if succeeded != 1 {
		t.Errorf("成功した更新の数 = %d、期待値 = 1", succeeded)
	}
	if n := countRows(t, db, "page_revisions", f.space.ID, ""); n != 1 {
		t.Errorf("リビジョンの数 = %d、期待値 = 1", n)
	}
}

// TestUpdateAPIPageUsecase_ExecuteWithWebPublishは、前提の照合の後にWeb画面の公開が割り込んでも、
// ページの行のロックで更新を待たせ、公開された内容を上書きせずに前提の不一致にすることを確かめる
func TestUpdateAPIPageUsecase_ExecuteWithWebPublish(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberScopes := apiTopicRegularMemberScopes
	f := setupCreateAPIPageFixture(t, db, "api-page-update-web-"+uuid.NewString()[:8], memberScopes)
	pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
		WithTitle("タイトル").WithBody("本文").Build()
	digest := findPageForTest(t, db, f.space.ID, 1).ContentDigest()

	// Web画面の公開は、スペースをロックせずにページの行を書き換える
	webTx, webPID := beginWebTx(t, db)
	if _, err := webTx.ExecContext(t.Context(), "UPDATE pages SET body = $1, modified_at = now() WHERE id = $2 AND space_id = $3", "Web画面で公開した本文", pageID, f.space.ID); err != nil {
		t.Fatalf("Web側の更新: %v", err)
	}

	resultCh := make(chan createAPIPageResult, 1)
	go func() {
		_, err := newUpdateAPIPageUC(db).Execute(t.Context(), UpdateAPIPageInput{
			Principal:       f.principal(memberScopes, []model.Scope{model.ScopePageWrite}),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			IfMatchDigests:  []string{digest},
			Body:            strPtr("APIで更新した本文"),
		})
		resultCh <- createAPIPageResult{err: err}
	}()

	waitCtx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	waitUntilBlockedBy(t, waitCtx, db, webPID, "FindPageByIDForUpdate", resultCh)

	if err := webTx.Commit(); err != nil {
		t.Fatalf("Web側トランザクションのコミット: %v", err)
	}
	var result createAPIPageResult
	select {
	case result = <-resultCh:
	case <-waitCtx.Done():
		t.Fatalf("競合解除後もAPIの更新が終了しなかった: %v", waitCtx.Err())
	}

	if ae := model.AsAppError(result.err); ae == nil || ae.Code != model.AppErrCodePreconditionFailed {
		t.Errorf("error = %v、期待値 = AppErrCodePreconditionFailed", result.err)
	}
	if got := findPageForTest(t, db, f.space.ID, 1).Body; got != "Web画面で公開した本文" {
		t.Errorf("本文 = %q、期待値 = Web画面で公開した本文", got)
	}
}

// execForTestはテストの前提を作るSQLを実行する
func execForTest(t *testing.T, db *sql.DB, sqlText string, args ...any) {
	t.Helper()

	if _, err := db.ExecContext(t.Context(), sqlText, args...); err != nil {
		t.Fatalf("SQLの実行に失敗: %v", err)
	}
}

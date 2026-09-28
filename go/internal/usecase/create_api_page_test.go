package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func newCreateAPIPageUC(db *sql.DB) *CreateAPIPageUsecase {
	q := query.New(db)
	pageRepo := repository.NewPageRepository(q)
	return NewCreateAPIPageUsecase(
		db,
		repository.NewSpaceRepository(q),
		pageRepo,
		repository.NewPageRevisionRepository(q),
		repository.NewPageEditorRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
		repository.NewAttachmentRepository(q),
		repository.NewPageAttachmentReferenceRepository(q),
		validator.NewPageCreateValidator(pageRepo),
	)
}

// createAPIPageFixtureはページ作成APIのUseCaseのテストで共有するフィクスチャ。
// トピックは1が公開、2が参加している非公開、3が参加していない非公開
type createAPIPageFixture struct {
	space         *model.Space
	spaceMemberID model.SpaceMemberID
	userID        model.UserID
	topicIDs      map[int32]model.TopicID
}

// setupCreateAPIPageFixtureは、UseCaseが自前でトランザクションを管理するため、フィクスチャを
// テストDBへ直接コミットして作成する。identifierは並行テスト間で一意にする
func setupCreateAPIPageFixture(t *testing.T, db *sql.DB, identifier string, memberScopes []model.Scope) createAPIPageFixture {
	t.Helper()

	atname := strings.ReplaceAll(identifier, "-", "_")
	userID := testutil.NewUserBuilderDB(t, db).WithAtname(atname).WithEmail(atname + "@example.com").Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier(identifier).Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithScopes(memberScopes).
		Build()

	publicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).WithNumber(1).WithName("公開").Build()
	joinedPrivateID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).WithNumber(2).WithName("参加している非公開").
		WithVisibility(int32(model.TopicVisibilityPrivate)).Build()
	testutil.NewTopicMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(joinedPrivateID).
		WithSpaceMemberID(spaceMemberID).
		WithScopes([]model.Scope{model.ScopeTopicRead, model.ScopePageWrite}).
		Build()
	notJoinedPrivateID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).WithNumber(3).WithName("参加していない非公開").
		WithVisibility(int32(model.TopicVisibilityPrivate)).Build()

	return createAPIPageFixture{
		space:         &model.Space{ID: spaceID, Identifier: model.SpaceIdentifier(identifier)},
		spaceMemberID: spaceMemberID,
		userID:        userID,
		topicIDs:      map[int32]model.TopicID{1: publicID, 2: joinedPrivateID, 3: notJoinedPrivateID},
	}
}

// principalはフィクスチャのメンバーが、tokenScopesを持つトークンで呼び出した主体を返す
func (f createAPIPageFixture) principal(memberScopes, tokenScopes []model.Scope) *model.APIPrincipal {
	return &model.APIPrincipal{
		User:        &model.User{ID: f.userID},
		Space:       f.space,
		SpaceMember: &model.SpaceMember{ID: f.spaceMemberID, SpaceID: f.space.ID, UserID: f.userID, Scopes: memberScopes, Active: true},
		TokenKind:   model.APITokenKindPersonalAccessToken,
		Scopes:      policy.ExpandAPITokenScopes(tokenScopes),
	}
}

// countRowsはspace_idで絞ったテーブルの行のうち、conditionを満たすものの数を返す
func countRows(t *testing.T, db *sql.DB, table string, spaceID model.SpaceID, condition string, args ...any) int {
	t.Helper()

	var n int
	sqlText := "SELECT count(*) FROM " + table + " WHERE space_id = $1"
	if condition != "" {
		sqlText += " AND " + condition
	}
	if err := db.QueryRowContext(t.Context(), sqlText, append([]any{spaceID}, args...)...).Scan(&n); err != nil {
		t.Fatalf("%s の行数の取得に失敗: %v", table, err)
	}
	return n
}

func TestCreateAPIPageUsecase_Execute(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberScopes := apiTopicRegularMemberScopes
	f := setupCreateAPIPageFixture(t, db, "api-page-create", memberScopes)

	// 本文はこのページ自身・既存のページ・存在しないページへのWikiリンクを含む
	existingID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("既存").Build()
	body := "[[新しいページ]] [[既存]] [[まだ無いページ]]"

	output, err := newCreateAPIPageUC(db).Execute(t.Context(), CreateAPIPageInput{
		Principal:       f.principal(memberScopes, []model.Scope{model.ScopePageWrite}),
		SpaceIdentifier: f.space.Identifier,
		TopicNumber:     1,
		Title:           "新しいページ",
		Body:            body,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	page := output.Page
	if page.Title == nil || *page.Title != "新しいページ" || page.Body != body || page.TopicID != f.topicIDs[1] {
		t.Errorf("ページ = %+v、期待値 = トピック1の「新しいページ」", page)
	}
	if page.PublishedAt == nil {
		t.Error("PublishedAt = nil、期待値 = 公開済み")
	}
	if output.Topic.ID != f.topicIDs[1] {
		t.Errorf("Topic.ID = %s、期待値 = %s", output.Topic.ID, f.topicIDs[1])
	}

	// 自分自身へのリンクで同じタイトルのページを別に作らず、存在しないページだけを下書きで作る
	if n := countRows(t, db, "pages", f.space.ID, "title = $2", "新しいページ"); n != 1 {
		t.Errorf("「新しいページ」の数 = %d、期待値 = 1", n)
	}
	if n := countRows(t, db, "pages", f.space.ID, "title = $2 AND published_at IS NULL", "まだ無いページ"); n != 1 {
		t.Errorf("リンク先として作成された「まだ無いページ」の数 = %d、期待値 = 1", n)
	}
	if len(page.LinkedPageIDs) != 3 || page.LinkedPageIDs[0] != page.ID || page.LinkedPageIDs[1] != existingID {
		t.Errorf("LinkedPageIDs = %v、期待値 = 自分・既存・作成したページ", page.LinkedPageIDs)
	}

	if n := countRows(t, db, "page_revisions", f.space.ID, "page_id = $2 AND space_member_id = $3", page.ID, f.spaceMemberID); n != 1 {
		t.Errorf("リビジョンの数 = %d、期待値 = 1", n)
	}
	if n := countRows(t, db, "page_editors", f.space.ID, "page_id = $2 AND space_member_id = $3", page.ID, f.spaceMemberID); n != 1 {
		t.Errorf("編集者の数 = %d、期待値 = 1", n)
	}
	if n := countRows(t, db, "draft_pages", f.space.ID, "page_id = $2", page.ID); n != 0 {
		t.Errorf("下書きの数 = %d、期待値 = 0", n)
	}
}

func TestCreateAPIPageUsecase_Execute_Authorization(t *testing.T) {
	t.Parallel()

	writeScopes := []model.Scope{model.ScopePageWrite}
	writeAndTopicScopes := []model.Scope{model.ScopePageWrite, model.ScopeTopicRead}
	tests := []struct {
		name         string
		memberScopes []model.Scope
		tokenScopes  []model.Scope
		identifier   model.SpaceIdentifier
		topicNumber  int32
		// wantFieldは422になる場合の問題のフィールド、wantCodeは未存在などのエラーのコード
		wantField string
		wantCode  model.AppErrorCode
	}{
		{
			name:         "トークンがtopic:readを持てば参加している非公開トピックに作成できる",
			memberScopes: apiTopicRegularMemberScopes,
			tokenScopes:  writeAndTopicScopes,
			topicNumber:  2,
		},
		{
			name:         "トークンがtopic:readを持たなければ参加している非公開トピックは見つからない",
			memberScopes: apiTopicRegularMemberScopes,
			tokenScopes:  writeScopes,
			topicNumber:  2,
			wantField:    "topic_number",
		},
		{
			name:         "参加していない非公開トピックは見つからない",
			memberScopes: apiTopicRegularMemberScopes,
			tokenScopes:  writeAndTopicScopes,
			topicNumber:  3,
			wantField:    "topic_number",
		},
		{
			name:         "存在しないトピックは見つからない",
			memberScopes: apiTopicRegularMemberScopes,
			tokenScopes:  writeAndTopicScopes,
			topicNumber:  99,
			wantField:    "topic_number",
		},
		{
			name:         "メンバーがpage:writeを持たなければ未存在",
			memberScopes: []model.Scope{model.ScopePageRead, model.ScopePersonalAccessTokenWrite},
			tokenScopes:  writeScopes,
			topicNumber:  1,
			wantCode:     model.AppErrCodeForbidden,
		},
		{
			name:         "トークンがpage:writeを持たなければ、メンバーが持っていても作成できない",
			memberScopes: apiTopicAdminMemberScopes,
			tokenScopes:  []model.Scope{model.ScopePageRead, model.ScopeTopicRead},
			topicNumber:  1,
			wantCode:     model.AppErrCodeForbidden,
		},
		{
			name:         "束縛先と異なるスペースは未存在",
			memberScopes: apiTopicRegularMemberScopes,
			tokenScopes:  writeScopes,
			identifier:   "api-page-create-auth-other",
			topicNumber:  1,
			wantCode:     model.AppErrCodeResourceNotFound,
		},
	}

	db := testutil.GetTestDB()
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupCreateAPIPageFixture(t, db, "api-page-create-auth-"+string(rune('a'+i)), tt.memberScopes)
			identifier := f.space.Identifier
			if tt.identifier != "" {
				identifier = tt.identifier
			}

			output, err := newCreateAPIPageUC(db).Execute(t.Context(), CreateAPIPageInput{
				Principal:       f.principal(tt.memberScopes, tt.tokenScopes),
				SpaceIdentifier: identifier,
				TopicNumber:     tt.topicNumber,
				Title:           "ページ",
				Body:            "本文",
			})

			switch {
			case tt.wantField != "":
				if ve := model.AsValidationError(err); ve == nil || !ve.HasFieldError(tt.wantField) {
					t.Errorf("error = %v、期待値 = %s のValidationError", err, tt.wantField)
				}
			case tt.wantCode != 0:
				if ae := model.AsAppError(err); ae == nil || ae.Code != tt.wantCode {
					t.Errorf("error = %v、期待値 = AppErrorCode %d", err, tt.wantCode)
				}
			default:
				if err != nil {
					t.Fatalf("Execute() error = %v", err)
				}
				if output.Page.TopicID != f.topicIDs[tt.topicNumber] {
					t.Errorf("TopicID = %s、期待値 = %s", output.Page.TopicID, f.topicIDs[tt.topicNumber])
				}
				return
			}

			if output != nil {
				t.Errorf("output = %+v、期待値 = nil", output)
			}
			if n := countRows(t, db, "pages", f.space.ID, ""); n != 0 {
				t.Errorf("ページの数 = %d、期待値 = 0", n)
			}
		})
	}
}

func TestCreateAPIPageUsecase_Execute_TitleConflict(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberScopes := apiTopicRegularMemberScopes
	tokenScopes := []model.Scope{model.ScopePageWrite}

	t.Run("公開済みのページと同じタイトルは422", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAPIPageFixture(t, db, "api-page-create-dup", memberScopes)
		testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("重複").Build()

		output, err := newCreateAPIPageUC(db).Execute(t.Context(), CreateAPIPageInput{
			Principal:       f.principal(memberScopes, tokenScopes),
			SpaceIdentifier: f.space.Identifier,
			TopicNumber:     1,
			Title:           "重複",
			Body:            "本文",
		})
		if output != nil {
			t.Errorf("output = %+v、期待値 = nil", output)
		}
		if ve := model.AsValidationError(err); ve == nil || !ve.HasFieldError("title") {
			t.Errorf("error = %v、期待値 = titleのValidationError", err)
		}
	})

	t.Run("別のトピックなら同じタイトルでも作成できる", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAPIPageFixture(t, db, "api-page-create-other-topic", memberScopes)
		testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[2]).WithNumber(1).WithTitle("重複").Build()

		if _, err := newCreateAPIPageUC(db).Execute(t.Context(), CreateAPIPageInput{
			Principal:       f.principal(memberScopes, tokenScopes),
			SpaceIdentifier: f.space.Identifier,
			TopicNumber:     1,
			Title:           "重複",
			Body:            "本文",
		}); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	})

	t.Run("Wikiリンクで作られた中身の無い未公開のページは論理削除して置き換える", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAPIPageFixture(t, db, "api-page-create-linked", memberScopes)
		linkedID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
			WithTitle("リンク先").WithBody("").WithUnpublished().Build()

		output, err := newCreateAPIPageUC(db).Execute(t.Context(), CreateAPIPageInput{
			Principal:       f.principal(memberScopes, tokenScopes),
			SpaceIdentifier: f.space.Identifier,
			TopicNumber:     1,
			Title:           "リンク先",
			Body:            "本文",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.Page.ID == linkedID || output.Page.Number != 2 {
			t.Errorf("ページ = %+v、期待値 = 番号2の新しいページ", output.Page)
		}
		if n := countRows(t, db, "pages", f.space.ID, "id = $2 AND discarded_at IS NOT NULL", linkedID); n != 1 {
			t.Error("未公開のページが論理削除されていない")
		}
	})
}

// TestCreateAPIPageUsecase_ExecuteConcurrentlyは、同じスペースへ異なるタイトルのページを
// 同時に作成しても、ページ番号の競合で失敗しないことを確かめる。
// 本文のWikiリンクが指す未作成のページも同じトランザクションで採番されるため、ページごとに
// 異なるリンク先と、全ページで共通のリンク先を持たせる
// createAPIPageResultは、別のゴルーチンで実行したページ作成の結果
type createAPIPageResult struct {
	output *CreateAPIPageOutput
	err    error
}

// executeCreateAPIPageAsyncは、Web側のトランザクションと競合させるためにページ作成を別の
// ゴルーチンで実行し、結果を受け取るチャネルを返す
func executeCreateAPIPageAsync(t *testing.T, db *sql.DB, input CreateAPIPageInput) <-chan createAPIPageResult {
	t.Helper()

	resultCh := make(chan createAPIPageResult, 1)
	go func() {
		output, err := newCreateAPIPageUC(db).Execute(t.Context(), input)
		resultCh <- createAPIPageResult{output: output, err: err}
	}()
	return resultCh
}

// beginWebTxは、スペースをロックせずにページを書き換えるWeb画面側のトランザクションを開始し、
// その接続のバックエンドのPIDとともに返す。トランザクションはテストの終わりにロールバックする
func beginWebTx(t *testing.T, db *sql.DB) (*sql.Tx, int) {
	t.Helper()

	webTx, err := db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatalf("Web側トランザクションの開始: %v", err)
	}
	t.Cleanup(func() { _ = webTx.Rollback() })
	var webPID int
	if err := webTx.QueryRowContext(t.Context(), "SELECT pg_backend_pid()").Scan(&webPID); err != nil {
		t.Fatalf("Web側接続のPID取得: %v", err)
	}
	return webTx, webPID
}

// waitUntilBlockedByは、queryNameのクエリを実行中の接続がwebPIDの接続のロックを待つまで待機する。
// 実行順に依存するsleepでは競合が成立しないことがあるため、ブロック状態をDBで確認する。
// 待つ前にページ作成が終わった場合と、ctxが期限切れになった場合はテストを失敗させる
func waitUntilBlockedBy(t *testing.T, ctx context.Context, db *sql.DB, webPID int, queryName string, resultCh <-chan createAPIPageResult) {
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
		)`, webPID, queryName).Scan(&blocked)
		if err != nil {
			t.Fatalf("APIのロック待ちの確認: %v", err)
		}
		if blocked {
			return
		}
		select {
		case result := <-resultCh:
			t.Fatalf("競合前にAPIが終了: output=%+v, err=%v", result.output, result.err)
		case <-ctx.Done():
			t.Fatalf("APIの %s がWeb側のトランザクションを待たなかった: %v", queryName, ctx.Err())
		case <-ticker.C:
		}
	}
}

func TestCreateAPIPageUsecase_ExecuteConcurrently(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberScopes := apiTopicRegularMemberScopes
	f := setupCreateAPIPageFixture(t, db, "api-page-cc-"+uuid.NewString()[:8], memberScopes)
	uc := newCreateAPIPageUC(db)

	// 1回の競合で番号を得られるのは1件だけのため、再試行の上限より多く並行させる
	const concurrency = 8
	outputs := make([]*CreateAPIPageOutput, concurrency)
	errs := make([]error, concurrency)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			outputs[i], errs[i] = uc.Execute(t.Context(), CreateAPIPageInput{
				Principal:       f.principal(memberScopes, []model.Scope{model.ScopePageWrite}),
				SpaceIdentifier: f.space.Identifier,
				TopicNumber:     1,
				Title:           fmt.Sprintf("日報%d", i),
				Body:            fmt.Sprintf("[[リンク先%d]] [[共通のリンク先]]", i),
			})
		}()
	}
	close(start)
	wg.Wait()

	for i := range concurrency {
		if errs[i] != nil {
			t.Fatalf("「日報%d」の作成で予期しないエラー: %v", i, errs[i])
		}
	}
	if n := countRows(t, db, "pages", f.space.ID, "published_at IS NOT NULL"); n != concurrency {
		t.Errorf("公開済みページの数 = %d、期待値 = %d", n, concurrency)
	}
	// ページごとのリンク先と共通のリンク先が1つずつ、未公開のページとして作られる
	if n := countRows(t, db, "pages", f.space.ID, "published_at IS NULL"); n != concurrency+1 {
		t.Errorf("未公開のページの数 = %d、期待値 = %d", n, concurrency+1)
	}
}

// TestCreateAPIPageUsecase_ExecuteWithWebConflictは、Web画面側がスペースをロックせずに
// ページを作る場合でも、APIが一意制約の競合からタイトルの検証をやり直して作成を終えられることを確かめる。
func TestCreateAPIPageUsecase_ExecuteWithWebConflict(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// Web側が未コミットで作るページ。webPublishedがfalseなら、Wikiリンクで作られるのと同じ
		// 本文が空の未公開ページになる
		webNumber    model.PageNumber
		webTitle     string
		webPublished bool
		apiTitle     string
		apiBody      string
		// wantTitleErrはAPIの作成がtitleのValidationErrorになること
		wantTitleErr bool
		wantNumber   model.PageNumber
		// wantLinkedToWebはAPIのページのリンク先がWeb側のページになること
		wantLinkedToWeb bool
		// wantWebDiscardedはWeb側のページが論理削除されて置き換えられること
		wantWebDiscarded bool
	}{
		{
			// 番号2はAPI自身のページに使わせ、リンク先が採番する番号3で競合させる
			name:       "リンク先の番号の競合は番号を取り直す",
			webNumber:  3,
			webTitle:   "Web側のリンク先",
			apiTitle:   "APIのページ",
			apiBody:    "[[APIのリンク先]]",
			wantNumber: 4,
		},
		{
			name:            "リンク先のタイトルの競合はWeb側のページにリンクする",
			webNumber:       100,
			webTitle:        "共通",
			apiTitle:        "APIのページ",
			apiBody:         "[[共通]]",
			wantNumber:      101,
			wantLinkedToWeb: true,
		},
		{
			name:         "公開済みのページとのタイトルの競合は重複になる",
			webNumber:    100,
			webTitle:     "APIのページ",
			webPublished: true,
			apiTitle:     "APIのページ",
			apiBody:      "本文",
			wantTitleErr: true,
		},
		{
			name:             "中身の無い未公開ページとのタイトルの競合は置き換える",
			webNumber:        100,
			webTitle:         "APIのページ",
			apiTitle:         "APIのページ",
			apiBody:          "本文",
			wantNumber:       101,
			wantWebDiscarded: true,
		},
	}

	db := testutil.GetTestDB()
	memberScopes := apiTopicRegularMemberScopes
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupCreateAPIPageFixture(t, db, "api-page-web-cc-"+uuid.NewString()[:8], memberScopes)
			testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("既存").Build()

			webTx, webPID := beginWebTx(t, db)
			webPage := testutil.NewPageBuilder(t, webTx).
				WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(tt.webNumber).WithTitle(tt.webTitle)
			if !tt.webPublished {
				webPage = webPage.WithBody("").WithUnpublished()
			}
			webPageID := webPage.Build()

			resultCh := executeCreateAPIPageAsync(t, db, CreateAPIPageInput{
				Principal:       f.principal(memberScopes, []model.Scope{model.ScopePageWrite}),
				SpaceIdentifier: f.space.Identifier,
				TopicNumber:     1,
				Title:           tt.apiTitle,
				Body:            tt.apiBody,
			})

			// APIのページ作成のINSERTがWeb側の未コミット行を待つまで、競合を解除しない
			waitCtx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			waitUntilBlockedBy(t, waitCtx, db, webPID, "CreateUnpublishedPage", resultCh)

			if err := webTx.Commit(); err != nil {
				t.Fatalf("Web側トランザクションのコミット: %v", err)
			}
			var result createAPIPageResult
			select {
			case result = <-resultCh:
			case <-waitCtx.Done():
				t.Fatalf("競合解除後もAPIの作成が終了しなかった: %v", waitCtx.Err())
			}

			if tt.wantTitleErr {
				if result.output != nil {
					t.Errorf("output = %+v、期待値 = nil", result.output)
				}
				if ve := model.AsValidationError(result.err); ve == nil || !ve.HasFieldError("title") {
					t.Errorf("error = %v、期待値 = titleのValidationError", result.err)
				}
				return
			}

			if result.err != nil {
				t.Fatalf("競合後のAPIの作成: %v", result.err)
			}
			page := result.output.Page
			if page.Number != tt.wantNumber {
				t.Errorf("APIページの番号 = %d、期待値 = %d", page.Number, tt.wantNumber)
			}
			if tt.wantLinkedToWeb && (len(page.LinkedPageIDs) != 1 || page.LinkedPageIDs[0] != webPageID) {
				t.Errorf("LinkedPageIDs = %v、期待値 = [%s]", page.LinkedPageIDs, webPageID)
			}
			discarded := countRows(t, db, "pages", f.space.ID, "id = $2 AND discarded_at IS NOT NULL", webPageID) == 1
			if discarded != tt.wantWebDiscarded {
				t.Errorf("Web側のページの論理削除 = %t、期待値 = %t", discarded, tt.wantWebDiscarded)
			}
		})
	}
}

// TestCreateAPIPageUsecase_ExecuteWhenReplacementIsPublishedは、APIが空の未公開ページを
// 置き換えると判断した後にWeb画面が公開しても、そのページを論理削除しないことを確かめる。
func TestCreateAPIPageUsecase_ExecuteWhenReplacementIsPublished(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberScopes := apiTopicRegularMemberScopes
	f := setupCreateAPIPageFixture(t, db, "api-page-publish-cc-"+uuid.NewString()[:8], memberScopes)
	pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).
		WithNumber(1).WithTitle("同じタイトル").WithBody("").WithUnpublished().Build()

	webTx, webPID := beginWebTx(t, db)
	if _, err := webTx.ExecContext(t.Context(), `
		UPDATE pages SET body = 'Web側の本文', published_at = now()
		WHERE id = $1 AND space_id = $2
	`, pageID, f.space.ID); err != nil {
		t.Fatalf("Web側での公開: %v", err)
	}

	resultCh := executeCreateAPIPageAsync(t, db, CreateAPIPageInput{
		Principal:       f.principal(memberScopes, []model.Scope{model.ScopePageWrite}),
		SpaceIdentifier: f.space.Identifier,
		TopicNumber:     1,
		Title:           "同じタイトル",
		Body:            "API側の本文",
	})

	// 条件付き更新がWeb側のページ行を待ったことを確認してから公開を確定する
	waitCtx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	waitUntilBlockedBy(t, waitCtx, db, webPID, "DiscardEmptyUnpublishedPageByID", resultCh)

	if err := webTx.Commit(); err != nil {
		t.Fatalf("Web側トランザクションのコミット: %v", err)
	}
	var result createAPIPageResult
	select {
	case result = <-resultCh:
	case <-waitCtx.Done():
		t.Fatalf("競合解除後もAPIが終了しなかった: %v", waitCtx.Err())
	}
	if result.output != nil {
		t.Errorf("output = %+v、期待値 = nil", result.output)
	}
	if ve := model.AsValidationError(result.err); ve == nil || !ve.HasFieldError("title") {
		t.Errorf("error = %v、期待値 = titleのValidationError", result.err)
	}
	if n := countRows(t, db, "pages", f.space.ID, "id = $2 AND body = $3 AND published_at IS NOT NULL AND discarded_at IS NULL", pageID, "Web側の本文"); n != 1 {
		t.Errorf("Web側が公開したページの数 = %d、期待値 = 1", n)
	}
	if n := countRows(t, db, "pages", f.space.ID, "id <> $2", pageID); n != 0 {
		t.Errorf("APIが作成したページの数 = %d、期待値 = 0", n)
	}
}

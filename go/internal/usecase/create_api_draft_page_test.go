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
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func newCreateAPIDraftPageUC(db *sql.DB) *CreateAPIDraftPageUsecase {
	q := query.New(db)
	return NewCreateAPIDraftPageUsecase(
		db,
		repository.NewSpaceRepository(q),
		repository.NewPageRepository(q),
		repository.NewPageEditorRepository(q),
		repository.NewDraftPageRepository(q),
		repository.NewDraftPageRevisionRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
		repository.NewAttachmentRepository(q),
		validator.NewAPIDraftPageCreateValidator(),
	)
}

func ptr[T any](v T) *T {
	return &v
}

// draftPageWriteScopesは下書きを作れるトークンのスコープ
var draftPageWriteScopes = []model.Scope{model.ScopeDraftPageWrite}

func TestCreateAPIDraftPageUsecase_Execute_ExistingPage(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	f := setupCreateAPIPageFixture(t, db, "api-draft-create-existing", memberRole)
	testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("ページ").WithBody("ページの本文").Build()
	testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(2).WithTitle("ページ2").WithBody("ページ2の本文").Build()
	testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(3).WithTitle("ページ3").WithBody("ページ3の本文").Build()

	tests := []struct {
		name       string
		pageNumber int32
		title      *string
		body       *string
		wantTitle  string
		wantBody   string
	}{
		{name: "タイトルと本文を省くとページの今の内容で作る", pageNumber: 1, wantTitle: "ページ", wantBody: "ページの本文"},
		{name: "本文だけを指定するとタイトルはページの今のタイトルになる", pageNumber: 2, body: ptr("新しい本文"), wantTitle: "ページ2", wantBody: "新しい本文"},
		{name: "タイトルと本文を指定するとその内容で作る", pageNumber: 3, title: ptr("新しいタイトル"), body: ptr(""), wantTitle: "新しいタイトル", wantBody: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := newCreateAPIDraftPageUC(db).Execute(t.Context(), CreateAPIDraftPageInput{
				Principal:       f.principal(memberRole, draftPageWriteScopes),
				SpaceIdentifier: f.space.Identifier,
				PageNumber:      ptr(tt.pageNumber),
				Title:           tt.title,
				Body:            tt.body,
			})
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}

			draft := output.DraftPage
			if draft.Title == nil || *draft.Title != tt.wantTitle || draft.Body != tt.wantBody {
				t.Errorf("下書き = (%v, %q)、期待値 = (%q, %q)", draft.Title, draft.Body, tt.wantTitle, tt.wantBody)
			}
			if draft.SpaceMemberID != f.spaceMemberID || draft.TopicID != f.topicIDs[1] {
				t.Errorf("下書きの持ち主・トピック = (%s, %s)、期待値 = (%s, %s)", draft.SpaceMemberID, draft.TopicID, f.spaceMemberID, f.topicIDs[1])
			}
			if draft.Page == nil || int32(draft.Page.Number) != tt.pageNumber {
				t.Errorf("下書きのページ = %+v、期待値 = ページ%d", draft.Page, tt.pageNumber)
			}
			if output.Topic.ID != f.topicIDs[1] {
				t.Errorf("Topic.ID = %s、期待値 = %s", output.Topic.ID, f.topicIDs[1])
			}
			if n := countRows(t, db, "draft_page_revisions", f.space.ID, "draft_page_id = $2 AND title = $3 AND body = $4", draft.ID, tt.wantTitle, tt.wantBody); n != 1 {
				t.Errorf("下書きと同じ内容のリビジョンの数 = %d、期待値 = 1", n)
			}
		})
	}

	// ページ自体は書き換えない
	if n := countRows(t, db, "pages", f.space.ID, "number = 3 AND title = 'ページ3' AND body = 'ページ3の本文'"); n != 1 {
		t.Errorf("元のままのページ3の数 = %d、期待値 = 1", n)
	}
}

func TestCreateAPIDraftPageUsecase_Execute_NewPage(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	f := setupCreateAPIPageFixture(t, db, "api-draft-create-new", memberRole)
	testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("既存").Build()

	t.Run("タイトルと本文を指定して作る", func(t *testing.T) {
		body := "[[既存]] [[まだ無いページ]]"
		output, err := newCreateAPIDraftPageUC(db).Execute(t.Context(), CreateAPIDraftPageInput{
			Principal:       f.principal(memberRole, draftPageWriteScopes),
			SpaceIdentifier: f.space.Identifier,
			TopicNumber:     ptr(int32(1)),
			Title:           ptr("新しいページ"),
			Body:            &body,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		draft := output.DraftPage
		if draft.Title == nil || *draft.Title != "新しいページ" || draft.Body != body {
			t.Errorf("下書き = (%v, %q)、期待値 = (\"新しいページ\", %q)", draft.Title, draft.Body, body)
		}

		// Web画面の作成と同じく、タイトルの無い空の未公開のページを作り、作成者を編集者にする
		page := draft.Page
		if page == nil || page.Title != nil || page.Body != "" || page.PublishedAt != nil || page.TopicID != f.topicIDs[1] {
			t.Errorf("ページ = %+v、期待値 = トピック1のタイトルの無い空の未公開のページ", page)
		}
		if n := countRows(t, db, "page_editors", f.space.ID, "page_id = $2 AND space_member_id = $3", page.ID, f.spaceMemberID); n != 1 {
			t.Errorf("編集者の数 = %d、期待値 = 1", n)
		}

		// Wikiリンクが指す存在しないページを未公開のページとして作る
		if n := countRows(t, db, "pages", f.space.ID, "title = $2 AND published_at IS NULL", "まだ無いページ"); n != 1 {
			t.Errorf("リンク先として作成された「まだ無いページ」の数 = %d、期待値 = 1", n)
		}
		if len(draft.LinkedPageIDs) != 2 {
			t.Errorf("LinkedPageIDs = %v、期待値 = 既存と作成したページ", draft.LinkedPageIDs)
		}
		if n := countRows(t, db, "draft_page_revisions", f.space.ID, "draft_page_id = $2 AND title = $3 AND body = $4", draft.ID, "新しいページ", body); n != 1 {
			t.Errorf("リビジョンの数 = %d、期待値 = 1", n)
		}
	})

	t.Run("タイトルと本文を省くとタイトルの無い空の下書きを作る", func(t *testing.T) {
		output, err := newCreateAPIDraftPageUC(db).Execute(t.Context(), CreateAPIDraftPageInput{
			Principal:       f.principal(memberRole, draftPageWriteScopes),
			SpaceIdentifier: f.space.Identifier,
			TopicNumber:     ptr(int32(1)),
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		draft := output.DraftPage
		if draft.Title != nil || draft.Body != "" {
			t.Errorf("下書き = (%v, %q)、期待値 = タイトルの無い空の下書き", draft.Title, draft.Body)
		}
		// 手動保存と同じく、タイトルの無い下書きのリビジョンは空のタイトルで記録する
		if n := countRows(t, db, "draft_page_revisions", f.space.ID, "draft_page_id = $2 AND title = '' AND body = ''", draft.ID); n != 1 {
			t.Errorf("リビジョンの数 = %d、期待値 = 1", n)
		}
	})
}

func TestCreateAPIDraftPageUsecase_Execute_AlreadyExists(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	f := setupCreateAPIPageFixture(t, db, "api-draft-create-exists", memberRole)
	pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("ページ").Build()
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(f.space.ID).WithPageID(pageID).WithSpaceMemberID(f.spaceMemberID).WithTopicID(f.topicIDs[1]).
		WithTitle("書きかけ").WithBody("書きかけの本文").Build()

	// 他のメンバーの下書きは、トークンの持ち主の下書きの作成を妨げない
	otherUserID := testutil.NewUserBuilderDB(t, db).WithAtname("api_draft_create_exists_other").WithEmail("api_draft_create_exists_other@example.com").Build()
	otherMemberID := testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(f.space.ID).WithUserID(otherUserID).Build()
	otherPageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(2).WithTitle("ページ2").Build()
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(f.space.ID).WithPageID(otherPageID).WithSpaceMemberID(otherMemberID).WithTopicID(f.topicIDs[1]).Build()

	output, err := newCreateAPIDraftPageUC(db).Execute(t.Context(), CreateAPIDraftPageInput{
		Principal:       f.principal(memberRole, draftPageWriteScopes),
		SpaceIdentifier: f.space.Identifier,
		PageNumber:      ptr(int32(1)),
		Body:            ptr("上書きしない"),
	})
	if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeAlreadyExists {
		t.Fatalf("エラー = %v、期待値 = AppErrCodeAlreadyExists", err)
	}
	if output != nil {
		t.Errorf("output = %+v、期待値 = nil", output)
	}
	if n := countRows(t, db, "draft_pages", f.space.ID, "page_id = $2 AND body = '書きかけの本文'", pageID); n != 1 {
		t.Errorf("元のままの下書きの数 = %d、期待値 = 1", n)
	}

	if _, err := newCreateAPIDraftPageUC(db).Execute(t.Context(), CreateAPIDraftPageInput{
		Principal:       f.principal(memberRole, draftPageWriteScopes),
		SpaceIdentifier: f.space.Identifier,
		PageNumber:      ptr(int32(2)),
	}); err != nil {
		t.Errorf("他のメンバーの下書きがあるページでのExecute()のエラー = %v", err)
	}
}

func TestCreateAPIDraftPageUsecase_Execute_Validation(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	f := setupCreateAPIPageFixture(t, db, "api-draft-create-validation", memberRole)
	testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("ページ").Build()
	trashedID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(2).WithTitle("ゴミ箱").Build()
	if _, err := db.ExecContext(t.Context(), "UPDATE pages SET trashed_at = NOW() WHERE id = $1 AND space_id = $2", trashedID, f.space.ID); err != nil {
		t.Fatalf("ページをゴミ箱に入れられない: %v", err)
	}
	testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[2]).WithNumber(3).WithTitle("非公開").Build()

	tests := []struct {
		name        string
		tokenScopes []model.Scope
		pageNumber  *int32
		topicNumber *int32
		// wantFieldsは問題のフィールド (空ならフォーム全体の問題)
		wantFields []string
	}{
		{name: "ページとトピックの両方を指定すると両方の問題", tokenScopes: draftPageWriteScopes, pageNumber: ptr(int32(1)), topicNumber: ptr(int32(1)), wantFields: []string{"page_number", "topic_number"}},
		{name: "どちらも指定しなければ全体の問題", tokenScopes: draftPageWriteScopes},
		{name: "存在しないページはpage_numberの問題", tokenScopes: draftPageWriteScopes, pageNumber: ptr(int32(99)), wantFields: []string{"page_number"}},
		{name: "ゴミ箱のページはpage_numberの問題", tokenScopes: draftPageWriteScopes, pageNumber: ptr(int32(2)), wantFields: []string{"page_number"}},
		{name: "トークンがtopic:readを持たなければ非公開トピックのページはpage_numberの問題", tokenScopes: draftPageWriteScopes, pageNumber: ptr(int32(3)), wantFields: []string{"page_number"}},
		{name: "存在しないトピックはtopic_numberの問題", tokenScopes: draftPageWriteScopes, topicNumber: ptr(int32(99)), wantFields: []string{"topic_number"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := newCreateAPIDraftPageUC(db).Execute(t.Context(), CreateAPIDraftPageInput{
				Principal:       f.principal(memberRole, tt.tokenScopes),
				SpaceIdentifier: f.space.Identifier,
				PageNumber:      tt.pageNumber,
				TopicNumber:     tt.topicNumber,
			})
			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatalf("エラー = %v、期待値 = ValidationError", err)
			}
			if output != nil {
				t.Errorf("output = %+v、期待値 = nil", output)
			}
			if len(tt.wantFields) == 0 && len(ve.Global) == 0 {
				t.Errorf("ValidationError = %+v、期待値 = 全体の問題", ve)
			}
			for _, field := range tt.wantFields {
				if !ve.HasFieldError(field) {
					t.Errorf("ValidationError = %+v、期待値 = %s の問題", ve, field)
				}
			}
		})
	}

	if n := countRows(t, db, "draft_pages", f.space.ID, ""); n != 0 {
		t.Errorf("下書きの数 = %d、期待値 = 0", n)
	}
	if n := countRows(t, db, "pages", f.space.ID, ""); n != 3 {
		t.Errorf("ページの数 = %d、期待値 = 3", n)
	}
}

func TestCreateAPIDraftPageUsecase_Execute_Authorization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		memberRole  model.SpaceRole
		tokenScopes []model.Scope
		identifier  model.SpaceIdentifier
		pageNumber  *int32
		topicNumber *int32
		// wantCodeはエラーのコード。0なら作成できる
		wantCode model.AppErrorCode
	}{
		{
			name:        "draft_page:writeだけのトークンで既存のページの下書きを作れる",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: draftPageWriteScopes,
			pageNumber:  ptr(int32(1)),
		},
		{
			name:        "draft_page:writeだけのトークンで新しいページの下書きを作れる",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: draftPageWriteScopes,
			topicNumber: ptr(int32(1)),
		},
		{
			name:        "page:writeだけのトークンでは既存のページの下書きを作れない",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: []model.Scope{model.ScopePageWrite},
			pageNumber:  ptr(int32(1)),
			wantCode:    model.AppErrCodeForbidden,
		},
		{
			name:        "page:writeだけのトークンでは新しいページの下書きを作れない",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: []model.Scope{model.ScopePageWrite},
			topicNumber: ptr(int32(1)),
			wantCode:    model.AppErrCodeForbidden,
		},
		{
			name:        "トークンがtopic:readを持てば非公開トピックに新しいページの下書きを作れる",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: append([]model.Scope{model.ScopeTopicRead}, draftPageWriteScopes...),
			topicNumber: ptr(int32(2)),
		},
		{
			name:        "メンバーが閲覧者なら、トークンが持っていても作れない",
			memberRole:  model.SpaceRoleViewer,
			tokenScopes: draftPageWriteScopes,
			pageNumber:  ptr(int32(1)),
			wantCode:    model.AppErrCodeForbidden,
		},
		{
			name:        "メンバーが閲覧者なら、トークンが持っていても新しいページの下書きを作れない",
			memberRole:  model.SpaceRoleViewer,
			tokenScopes: draftPageWriteScopes,
			topicNumber: ptr(int32(1)),
			wantCode:    model.AppErrCodeForbidden,
		},
		{
			name:        "トークンが属さないスペースは未存在",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: draftPageWriteScopes,
			identifier:  "api-draft-create-auth-other",
			pageNumber:  ptr(int32(1)),
			wantCode:    model.AppErrCodeResourceNotFound,
		},
	}

	db := testutil.GetTestDB()
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupCreateAPIPageFixture(t, db, "api-draft-create-auth-"+string(rune('a'+i)), tt.memberRole)
			testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("ページ").Build()
			identifier := f.space.Identifier
			if tt.identifier != "" {
				identifier = tt.identifier
			}

			output, err := newCreateAPIDraftPageUC(db).Execute(t.Context(), CreateAPIDraftPageInput{
				Principal:       f.principal(tt.memberRole, tt.tokenScopes),
				SpaceIdentifier: identifier,
				PageNumber:      tt.pageNumber,
				TopicNumber:     tt.topicNumber,
			})

			if tt.wantCode == 0 {
				if err != nil {
					t.Fatalf("Execute()のエラー = %v", err)
				}
				if n := countRows(t, db, "draft_pages", f.space.ID, "id = $2", output.DraftPage.ID); n != 1 {
					t.Errorf("作成した下書きの数 = %d、期待値 = 1", n)
				}
				return
			}

			if ae := model.AsAppError(err); ae == nil || ae.Code != tt.wantCode {
				t.Errorf("エラー = %v、期待値 = AppErrorCode %d", err, tt.wantCode)
			}
			if output != nil {
				t.Errorf("output = %+v、期待値 = nil", output)
			}
			if n := countRows(t, db, "draft_pages", f.space.ID, ""); n != 0 {
				t.Errorf("下書きの数 = %d、期待値 = 0", n)
			}
			if n := countRows(t, db, "pages", f.space.ID, ""); n != 1 {
				t.Errorf("ページの数 = %d、期待値 = 1", n)
			}
		})
	}
}

// Web保存が先に下書きを作った場合も、APIがリンク先を先に作って循環待ちにしないことを確かめる。
func TestCreateAPIDraftPageUsecase_Execute_ConcurrentWebSave(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	f := setupCreateAPIPageFixture(t, db, "api-draft-concurrent-web", memberRole)
	pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("既存").Build()
	webTx, webPID := beginWebTx(t, db)
	q := query.New(db).WithTx(webTx)
	draftRepo := repository.NewDraftPageRepository(q)
	now := time.Now()
	_, err := draftRepo.CreateIfNotExists(t.Context(), repository.CreateDraftPageInput{
		SpaceID: f.space.ID, PageID: pageID, SpaceMemberID: f.spaceMemberID, TopicID: f.topicIDs[1], ModifiedAt: now,
	})
	if err != nil {
		t.Fatalf("Web側の下書き作成: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	type result struct {
		output *CreateAPIDraftPageOutput
		err    error
	}
	results := make(chan result, 1)
	go func() {
		output, err := newCreateAPIDraftPageUC(db).Execute(ctx, CreateAPIDraftPageInput{
			Principal: f.principal(memberRole, draftPageWriteScopes), SpaceIdentifier: f.space.Identifier,
			PageNumber: ptr(int32(1)), Body: ptr("[[共有リンク先]] [[APIだけのリンク先]]"),
		})
		results <- result{output: output, err: err}
	}()

	// APIの下書きINSERTがWebの未コミットの下書きを待つまで、Webのリンク解決を進めない。
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var blocked bool
		err := db.QueryRowContext(ctx, `SELECT EXISTS (
   SELECT 1 FROM pg_stat_activity
   WHERE datname = current_database()
     AND $1 = ANY(pg_blocking_pids(pid))
     AND query LIKE '%CreateDraftPageIfNotExists%'
  )`, webPID).Scan(&blocked)
		if err != nil {
			t.Fatalf("APIのロック待ちの確認: %v", err)
		}
		if blocked {
			break
		}
		select {
		case r := <-results:
			t.Fatalf("競合前にAPIが終了: output=%+v, err=%v", r.output, r.err)
		case <-ctx.Done():
			t.Fatalf("APIがWeb側の下書きを待たなかった: %v", ctx.Err())
		case <-ticker.C:
		}
	}

	// 自動保存・手動保存の共通処理でリンク先を解決し、下書きを更新する。
	webDraft, err := saveDraftPageContent(ctx, saveDraftPageContentInput{
		SpaceID: f.space.ID, PageID: pageID, SpaceMemberID: f.spaceMemberID, TopicID: f.topicIDs[1],
		Title: ptr("Webの下書き"), Body: "[[共有リンク先]]", CurrentTopicName: "公開",
	}, now, draftRepo, repository.NewPageRepository(q), repository.NewPageEditorRepository(q), repository.NewTopicRepository(q))
	if err != nil {
		t.Fatalf("Web側の保存: %v", err)
	}
	if err := webTx.Commit(); err != nil {
		t.Fatalf("Web側のコミット: %v", err)
	}

	var r result
	select {
	case r = <-results:
	case <-ctx.Done():
		t.Fatalf("Web保存後もAPIが終了しなかった: %v", ctx.Err())
	}
	if ae := model.AsAppError(r.err); ae == nil || ae.Code != model.AppErrCodeAlreadyExists {
		t.Fatalf("APIのエラー = %v、期待値 = AppErrCodeAlreadyExists (409)", r.err)
	}
	if r.output != nil {
		t.Errorf("APIのoutput = %+v、期待値 = nil", r.output)
	}
	if len(webDraft.LinkedPageIDs) != 1 {
		t.Errorf("Web下書きのリンク先 = %v、期待値 = 共有リンク先1件", webDraft.LinkedPageIDs)
	}
	if n := countRows(t, db, "draft_pages", f.space.ID, "page_id = $2 AND title = 'Webの下書き' AND body = '[[共有リンク先]]'", pageID); n != 1 {
		t.Errorf("Webの内容を保持した下書きの数 = %d、期待値 = 1", n)
	}
	if n := countRows(t, db, "pages", f.space.ID, ""); n != 2 {
		t.Errorf("ページ数 = %d、期待値 = 既存と共有リンク先の2件", n)
	}
	if n := countRows(t, db, "pages", f.space.ID, "title = '共有リンク先' AND published_at IS NULL"); n != 1 {
		t.Errorf("未公開の共有リンク先の数 = %d、期待値 = 1", n)
	}
	if n := countRows(t, db, "pages", f.space.ID, "title = 'APIだけのリンク先'"); n != 0 {
		t.Errorf("APIだけのリンク先の数 = %d、期待値 = 0", n)
	}
	if n := countRows(t, db, "page_editors", f.space.ID, ""); n != 1 {
		t.Errorf("編集者レコード数 = %d、期待値 = Webが作った共有リンク先の1件", n)
	}
	if n := countRows(t, db, "draft_page_revisions", f.space.ID, ""); n != 0 {
		t.Errorf("リビジョン数 = %d、期待値 = 0", n)
	}
}

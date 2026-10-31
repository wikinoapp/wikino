package usecase

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func newCreateAPIPageBodyEditUC(db *sql.DB) *CreateAPIPageBodyEditUsecase {
	q := query.New(db)
	pageRepo := repository.NewPageRepository(q)
	return NewCreateAPIPageBodyEditUsecase(
		db,
		repository.NewSpaceRepository(q),
		pageRepo,
		repository.NewPageRevisionRepository(q),
		repository.NewPageEditorRepository(q),
		repository.NewDraftPageRepository(q),
		repository.NewSuggestionPageRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
		repository.NewAttachmentRepository(q),
		repository.NewPageAttachmentReferenceRepository(q),
		validator.NewAPIPageUpdateValidator(pageRepo),
	)
}

func TestCreateAPIPageBodyEditUsecase_Execute(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	tokenScopes := []model.Scope{model.ScopePageWrite}

	t.Run("1件の編集で本文を置換し、リビジョンを作って下書きと公開日時に触れない", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAPIPageFixture(t, db, "api-page-body-edit", memberRole)
		publishedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
			WithTitle("タイトル").WithBody("- [ ] **6b-1**\n- [ ] **6b-2**").WithPublishedAt(publishedAt).Build()
		testutil.NewDraftPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithPageID(pageID).WithTopicID(f.topicIDs[1]).
			WithSpaceMemberID(f.spaceMemberID).WithTitle("タイトル").WithBody("下書きの本文").Build()
		before := findPageForTest(t, db, f.space.ID, 1)

		output, err := newCreateAPIPageBodyEditUC(db).Execute(t.Context(), CreateAPIPageBodyEditInput{
			Principal:       f.principal(memberRole, tokenScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			Edits:           []model.BodyEdit{{OldStr: "- [ ] **6b-1**", NewStr: "- [x] **6b-1**"}},
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		page := output.Page
		if page.ID != pageID || page.Title == nil || *page.Title != "タイトル" || page.Body != "- [x] **6b-1**\n- [ ] **6b-2**" {
			t.Errorf("ページ = %+v、期待値 = 本文を置換したページ", page)
		}
		if !page.ModifiedAt.After(before.ModifiedAt) {
			t.Errorf("ModifiedAt = %s、期待値 = %s より後", page.ModifiedAt, before.ModifiedAt)
		}
		if page.PublishedAt == nil || !page.PublishedAt.Equal(publishedAt) {
			t.Errorf("PublishedAt = %v、期待値 = %s のまま", page.PublishedAt, publishedAt)
		}
		if output.Topic.ID != f.topicIDs[1] {
			t.Errorf("Topic.ID = %s、期待値 = %s", output.Topic.ID, f.topicIDs[1])
		}
		if n := countRows(t, db, "page_revisions", f.space.ID, "page_id = $2 AND space_member_id = $3 AND body = $4", pageID, f.spaceMemberID, page.Body); n != 1 {
			t.Errorf("リビジョンの数 = %d、期待値 = 1", n)
		}
		if n := countRows(t, db, "draft_pages", f.space.ID, "page_id = $2 AND body = $3", pageID, "下書きの本文"); n != 1 {
			t.Errorf("下書きの数 = %d、期待値 = 1 (置換前のまま残る)", n)
		}
	})

	t.Run("複数の編集を順に、前の編集の結果へ適用する", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAPIPageFixture(t, db, "api-page-body-edit-multi", memberRole)
		testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
			WithTitle("タイトル").WithBody("## 実装判断ログ\n\n## タスクリスト\n").Build()

		output, err := newCreateAPIPageBodyEditUC(db).Execute(t.Context(), CreateAPIPageBodyEditInput{
			Principal:       f.principal(memberRole, tokenScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			Edits: []model.BodyEdit{
				{OldStr: "## 実装判断ログ\n", NewStr: "## 実装判断ログ\n\n### タスク1\n"},
				{OldStr: "### タスク1\n", NewStr: "### タスク1: 追加\n"},
			},
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if want := "## 実装判断ログ\n\n### タスク1: 追加\n\n## タスクリスト\n"; output.Page.Body != want {
			t.Errorf("本文 = %q、期待値 = %q", output.Page.Body, want)
		}
		if n := countRows(t, db, "page_revisions", f.space.ID, ""); n != 1 {
			t.Errorf("リビジョンの数 = %d、期待値 = 1 (編集の数ではなく1回の更新)", n)
		}
	})

	t.Run("replace_allなら一致した箇所をすべて置換する", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAPIPageFixture(t, db, "api-page-body-edit-all", memberRole)
		testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
			WithTitle("タイトル").WithBody("foo bar foo").Build()

		output, err := newCreateAPIPageBodyEditUC(db).Execute(t.Context(), CreateAPIPageBodyEditInput{
			Principal:       f.principal(memberRole, tokenScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			Edits:           []model.BodyEdit{{OldStr: "foo", NewStr: "baz", ReplaceAll: true}},
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.Page.Body != "baz bar baz" {
			t.Errorf("本文 = %q、期待値 = %q", output.Page.Body, "baz bar baz")
		}
	})

	t.Run("置換で足したWikiリンクの先のページを未公開で作る", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAPIPageFixture(t, db, "api-page-body-edit-link", memberRole)
		testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
			WithTitle("タイトル").WithBody("関連: なし").Build()

		output, err := newCreateAPIPageBodyEditUC(db).Execute(t.Context(), CreateAPIPageBodyEditInput{
			Principal:       f.principal(memberRole, tokenScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			Edits:           []model.BodyEdit{{OldStr: "なし", NewStr: "[[まだ無いページ]]"}},
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if n := countRows(t, db, "pages", f.space.ID, "title = $2 AND published_at IS NULL", "まだ無いページ"); n != 1 {
			t.Errorf("リンク先として作成された「まだ無いページ」の数 = %d、期待値 = 1", n)
		}
		if len(output.Page.LinkedPageIDs) != 1 {
			t.Errorf("LinkedPageIDs = %v、期待値 = 作成したページ1件", output.Page.LinkedPageIDs)
		}
	})

	t.Run("置換しても本文が変わらなければ何も書き込まない", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAPIPageFixture(t, db, "api-page-body-edit-noop", memberRole)
		testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
			WithTitle("タイトル").WithBody("本文").Build()
		before := findPageForTest(t, db, f.space.ID, 1)

		output, err := newCreateAPIPageBodyEditUC(db).Execute(t.Context(), CreateAPIPageBodyEditInput{
			Principal:       f.principal(memberRole, tokenScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			Edits:           []model.BodyEdit{{OldStr: "本文", NewStr: "本文"}},
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.Page.ContentDigest() != before.ContentDigest() {
			t.Error("本文を変えない置換で版が変わった")
		}
		if n := countRows(t, db, "page_revisions", f.space.ID, ""); n != 0 {
			t.Errorf("リビジョンの数 = %d、期待値 = 0", n)
		}
	})

	t.Run("本文のCRLFをLFとして照合し、置換した本文をLFで保存する", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAPIPageFixture(t, db, "api-page-body-edit-crlf", memberRole)
		testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
			WithTitle("タイトル").WithBody("1行目\r\n2行目\r\n").Build()

		output, err := newCreateAPIPageBodyEditUC(db).Execute(t.Context(), CreateAPIPageBodyEditInput{
			Principal:       f.principal(memberRole, tokenScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			Edits:           []model.BodyEdit{{OldStr: "1行目\n2行目", NewStr: "一行目\n二行目"}},
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if want := "一行目\n二行目\n"; output.Page.Body != want {
			t.Errorf("本文 = %q、期待値 = %q", output.Page.Body, want)
		}
	})
}

func TestCreateAPIPageBodyEditUsecase_Execute_Mismatch(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	tokenScopes := []model.Scope{model.ScopePageWrite}

	tests := []struct {
		name        string
		edits       []model.BodyEdit
		wantField   string
		wantMessage string
	}{
		{
			name: "一致が0件の編集があれば、前の編集も適用せずに何番目かを返す",
			edits: []model.BodyEdit{
				{OldStr: "foo", NewStr: "x"},
				{OldStr: "無い", NewStr: "y"},
			},
			wantField:   "edits/1/old_str",
			wantMessage: "一致する箇所が本文にありません",
		},
		{
			name:        "replace_allでなく一致が複数件なら、一致の件数を返す",
			edits:       []model.BodyEdit{{OldStr: "bar", NewStr: "x"}},
			wantField:   "edits/0/old_str",
			wantMessage: "2件あります",
		},
		{
			name:        "大文字小文字を区別する",
			edits:       []model.BodyEdit{{OldStr: "FOO", NewStr: "x"}},
			wantField:   "edits/0/old_str",
			wantMessage: "一致する箇所が本文にありません",
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupCreateAPIPageFixture(t, db, "api-page-body-edit-mis-"+string(rune('a'+i)), memberRole)
			pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
				WithTitle("タイトル").WithBody("foo bar bar").Build()

			output, err := newCreateAPIPageBodyEditUC(db).Execute(t.Context(), CreateAPIPageBodyEditInput{
				Principal:       f.principal(memberRole, tokenScopes),
				SpaceIdentifier: f.space.Identifier,
				PageNumber:      1,
				Edits:           tt.edits,
			})

			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatalf("error = %v、期待値 = ValidationError", err)
			}
			if len(ve.Fields) != 1 || len(ve.GetFieldErrors(tt.wantField)) != 1 {
				t.Fatalf("Fields = %v、期待値 = %s だけのエラー", ve.Fields, tt.wantField)
			}
			if msg := ve.GetFieldErrors(tt.wantField)[0]; !strings.Contains(msg, tt.wantMessage) {
				t.Errorf("メッセージ = %q、期待値 = %q を含む", msg, tt.wantMessage)
			}
			if output != nil {
				t.Errorf("output = %+v、期待値 = nil", output)
			}
			if n := countRows(t, db, "pages", f.space.ID, "id = $2 AND body = $3", pageID, "foo bar bar"); n != 1 {
				t.Error("ページが更新された")
			}
			if n := countRows(t, db, "page_revisions", f.space.ID, ""); n != 0 {
				t.Errorf("リビジョンの数 = %d、期待値 = 0", n)
			}
		})
	}
}

func TestCreateAPIPageBodyEditUsecase_Execute_Precondition(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	tokenScopes := []model.Scope{model.ScopePageWrite}

	tests := []struct {
		name       string
		hasIfMatch bool
		digests    func(current string) []string
		any        bool
		wantFailed bool
	}{
		{name: "If-Matchが無ければ版を照合せずに置換する"},
		{name: "今の版と一致すれば置換する", hasIfMatch: true, digests: func(current string) []string { return []string{current} }},
		{name: "`*` なら版を問わず置換する", hasIfMatch: true, any: true},
		{name: "今の版と一致しなければ412", hasIfMatch: true, digests: func(string) []string { return []string{"古い版"} }, wantFailed: true},
		{name: "If-Matchがあって候補が無ければ412", hasIfMatch: true, wantFailed: true},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupCreateAPIPageFixture(t, db, "api-page-body-edit-pre-"+string(rune('a'+i)), memberRole)
			testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
				WithTitle("タイトル").WithBody("本文").Build()
			current := findPageForTest(t, db, f.space.ID, 1).ContentDigest()
			var digests []string
			if tt.digests != nil {
				digests = tt.digests(current)
			}

			output, err := newCreateAPIPageBodyEditUC(db).Execute(t.Context(), CreateAPIPageBodyEditInput{
				Principal:       f.principal(memberRole, tokenScopes),
				SpaceIdentifier: f.space.Identifier,
				PageNumber:      1,
				HasIfMatch:      tt.hasIfMatch,
				IfMatchDigests:  digests,
				IfMatchAny:      tt.any,
				Edits:           []model.BodyEdit{{OldStr: "本文", NewStr: "新しい本文"}},
			})

			if !tt.wantFailed {
				if err != nil {
					t.Fatalf("Execute() error = %v", err)
				}
				if output.Page.Body != "新しい本文" {
					t.Errorf("本文 = %q、期待値 = 新しい本文", output.Page.Body)
				}
				return
			}
			if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodePreconditionFailed {
				t.Errorf("error = %v、期待値 = AppErrCodePreconditionFailed", err)
			}
			if got := findPageForTest(t, db, f.space.ID, 1).Body; got != "本文" {
				t.Errorf("本文 = %q、期待値 = 置換されない", got)
			}
		})
	}
}

func TestCreateAPIPageBodyEditUsecase_Execute_Unpublished(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	tokenScopes := []model.Scope{model.ScopePageWrite}

	t.Run("未公開のページの本文を置換すると公開する", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAPIPageFixture(t, db, "api-page-body-edit-unpub", memberRole)
		pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
			WithTitle("リンク先").WithBody("書きかけ").WithUnpublished().Build()

		output, err := newCreateAPIPageBodyEditUC(db).Execute(t.Context(), CreateAPIPageBodyEditInput{
			Principal:       f.principal(memberRole, tokenScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			Edits:           []model.BodyEdit{{OldStr: "書きかけ", NewStr: "書き終えた"}},
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if !output.Page.IsPublished() || output.Page.Body != "書き終えた" {
			t.Errorf("ページ = %+v、期待値 = 本文を置換して公開したページ", output.Page)
		}
		if n := countRows(t, db, "pages", f.space.ID, "id = $2 AND published_at IS NOT NULL", pageID); n != 1 {
			t.Error("DBのページが公開されていない")
		}
	})

	t.Run("タイトルの無い未公開のページは422", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAPIPageFixture(t, db, "api-page-body-edit-untitled", memberRole)
		testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
			WithNilTitle().WithBody("書きかけ").WithUnpublished().Build()

		_, err := newCreateAPIPageBodyEditUC(db).Execute(t.Context(), CreateAPIPageBodyEditInput{
			Principal:       f.principal(memberRole, tokenScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			Edits:           []model.BodyEdit{{OldStr: "書きかけ", NewStr: "書き終えた"}},
		})
		if ve := model.AsValidationError(err); ve == nil || !ve.HasFieldError("title") {
			t.Errorf("error = %v、期待値 = titleのValidationError", err)
		}
	})
}

func TestCreateAPIPageBodyEditUsecase_Execute_Authorization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		memberRole  model.SpaceRole
		tokenScopes []model.Scope
		topicNumber int32
		wantCode    model.AppErrorCode
	}{
		{
			name:        "トークンがtopic:readを持てば参加している非公開トピックのページを置換できる",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: []model.Scope{model.ScopePageWrite, model.ScopeTopicRead},
			topicNumber: 2,
		},
		{
			name:        "トークンがtopic:readを持たなければ非公開トピックのページは未存在",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: []model.Scope{model.ScopePageWrite},
			topicNumber: 2,
			wantCode:    model.AppErrCodeResourceNotFound,
		},
		{
			name:        "トークンがpage:writeを持たなければ、メンバーが持っていても置換できない",
			memberRole:  apiTopicAdminMemberRole,
			tokenScopes: []model.Scope{model.ScopePageRead, model.ScopeTopicRead},
			topicNumber: 1,
			wantCode:    model.AppErrCodeForbidden,
		},
		{
			name:        "メンバーが閲覧者なら置換できない",
			memberRole:  model.SpaceRoleViewer,
			tokenScopes: []model.Scope{model.ScopePageWrite},
			topicNumber: 1,
			wantCode:    model.AppErrCodeForbidden,
		},
	}

	db := testutil.GetTestDB()
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupCreateAPIPageFixture(t, db, "api-page-body-edit-auth-"+string(rune('a'+i)), tt.memberRole)
			pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[tt.topicNumber]).WithNumber(1).
				WithTitle("タイトル").WithBody("本文").Build()

			output, err := newCreateAPIPageBodyEditUC(db).Execute(t.Context(), CreateAPIPageBodyEditInput{
				Principal:       f.principal(tt.memberRole, tt.tokenScopes),
				SpaceIdentifier: f.space.Identifier,
				PageNumber:      1,
				Edits:           []model.BodyEdit{{OldStr: "本文", NewStr: "新しい本文"}},
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

func TestCreateAPIPageBodyEditUsecase_Execute_BodySize(t *testing.T) {
	t.Parallel()
	db := testutil.GetTestDB()
	for i, lang := range []string{"ja", "en"} {
		t.Run(lang, func(t *testing.T) {
			t.Parallel()
			f := setupCreateAPIPageFixture(t, db, "api-page-body-size-"+lang, apiTopicRegularMemberRole)
			testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("タイトル").WithBody("a").Build()
			before := findPageForTest(t, db, f.space.ID, 1)
			ctx := i18n.SetLocale(t.Context(), lang)
			output, err := newCreateAPIPageBodyEditUC(db).Execute(ctx, CreateAPIPageBodyEditInput{
				Principal:       f.principal(apiTopicRegularMemberRole, []model.Scope{model.ScopePageWrite}),
				SpaceIdentifier: f.space.Identifier,
				PageNumber:      1,
				Edits: []model.BodyEdit{
					{OldStr: "a", NewStr: "[[作らないページ]]"},
					{OldStr: "[[作らないページ]]", NewStr: strings.Repeat("b", model.BodyEditMaxBodyBytes+1)},
					{OldStr: "b", NewStr: "", ReplaceAll: true},
				},
			})
			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatalf("エラー = %v、期待値 = ValidationError", err)
			}
			messages := ve.GetFieldErrors("edits/1/new_str")
			if len(ve.Fields) != 1 || len(messages) != 1 {
				t.Fatalf("Fields = %v、期待値 = edits/1/new_strのエラー1件", ve.Fields)
			}
			if !strings.Contains(messages[0], "1048576") || (i == 0 && !strings.Contains(messages[0], "バイト")) || (i == 1 && !strings.Contains(messages[0], "bytes")) {
				t.Errorf("メッセージ = %q、期待値 = ロケールに応じた最大バイト数の案内", messages[0])
			}
			if output != nil {
				t.Errorf("output = %+v、期待値 = nil", output)
			}
			after := findPageForTest(t, db, f.space.ID, 1)
			if after.Body != before.Body || after.ContentDigest() != before.ContentDigest() {
				t.Error("拒否した編集で本文または版が変わった")
			}
			if n := countRows(t, db, "page_revisions", f.space.ID, ""); n != 0 {
				t.Errorf("リビジョンの数 = %d、期待値 = 0", n)
			}
			if n := countRows(t, db, "pages", f.space.ID, ""); n != 1 {
				t.Errorf("ページの数 = %d、期待値 = 1 (Wikiリンク先も作らない)", n)
			}
		})
	}
}

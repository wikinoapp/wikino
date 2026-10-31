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
		repository.NewDraftPageRepository(q),
		repository.NewSuggestionPageRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
		repository.NewAttachmentRepository(q),
		repository.NewPageAttachmentReferenceRepository(q),
		validator.NewPageCreateValidator(pageRepo),
	)
}

// createAPIPageFixtureはページ作成APIのUseCaseのテストで共有するフィクスチャ。
// トピックは1が公開、2がトピック編集者として参加している非公開、3が参加していない非公開
type createAPIPageFixture struct {
	space         *model.Space
	spaceMemberID model.SpaceMemberID
	userID        model.UserID
	topicIDs      map[int32]model.TopicID
}

// setupCreateAPIPageFixtureは、UseCaseが自前でトランザクションを管理するため、フィクスチャを
// テストDBへ直接コミットして作成する。identifierは並行テスト間で一意にする
func setupCreateAPIPageFixture(t *testing.T, db *sql.DB, identifier string, memberRole model.SpaceRole) createAPIPageFixture {
	t.Helper()

	atname := strings.ReplaceAll(identifier, "-", "_")
	userID := testutil.NewUserBuilderDB(t, db).WithAtname(atname).WithEmail(atname + "@example.com").Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier(identifier).Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithRole(memberRole).
		Build()

	publicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).WithNumber(1).WithName("公開").Build()
	joinedPrivateID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).WithNumber(2).WithName("参加している非公開").
		WithVisibility(int32(model.TopicVisibilityPrivate)).Build()
	testutil.NewTopicMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(joinedPrivateID).
		WithSpaceMemberID(spaceMemberID).
		WithRole(model.TopicRoleEditor).
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
func (f createAPIPageFixture) principal(memberRole model.SpaceRole, tokenScopes []model.Scope) *model.APIPrincipal {
	return &model.APIPrincipal{
		User:        &model.User{ID: f.userID},
		Space:       f.space,
		SpaceMember: &model.SpaceMember{ID: f.spaceMemberID, SpaceID: f.space.ID, UserID: f.userID, Role: memberRole, Active: true},
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
	memberRole := apiTopicRegularMemberRole
	f := setupCreateAPIPageFixture(t, db, "api-page-create", memberRole)

	// 本文はこのページ自身・既存のページ・存在しないページへのWikiリンクを含む
	existingID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("既存").Build()
	body := "[[新しいページ]] [[既存]] [[まだ無いページ]]"

	output, err := newCreateAPIPageUC(db).Execute(t.Context(), CreateAPIPageInput{
		Principal:       f.principal(memberRole, []model.Scope{model.ScopePageWrite}),
		SpaceIdentifier: f.space.Identifier,
		TopicNumber:     1,
		Title:           "新しいページ",
		Body:            body,
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
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
		name        string
		memberRole  model.SpaceRole
		tokenScopes []model.Scope
		identifier  model.SpaceIdentifier
		topicNumber int32
		// wantFieldは422になる場合の問題のフィールド、wantCodeは未存在などのエラーのコード
		wantField string
		wantCode  model.AppErrorCode
	}{
		{
			name:        "トークンがtopic:readを持てば参加している非公開トピックに作成できる",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: writeAndTopicScopes,
			topicNumber: 2,
		},
		{
			name:        "トークンがtopic:readを持たなければ参加している非公開トピックは見つからない",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: writeScopes,
			topicNumber: 2,
			wantField:   "topic_number",
		},
		{
			// どのスペースのロールもtopic:readを持つため、参加していない非公開トピックも開ける
			name:        "トークンがtopic:readを持てば参加していない非公開トピックにも作成できる",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: writeAndTopicScopes,
			topicNumber: 3,
		},
		{
			name:        "存在しないトピックは見つからない",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: writeAndTopicScopes,
			topicNumber: 99,
			wantField:   "topic_number",
		},
		{
			name:        "メンバーが閲覧者でpage:writeを持たなければ権限不足になる",
			memberRole:  model.SpaceRoleViewer,
			tokenScopes: writeScopes,
			topicNumber: 1,
			wantCode:    model.AppErrCodeForbidden,
		},
		{
			name:        "トークンがpage:writeを持たなければ、メンバーが持っていても作成できない",
			memberRole:  apiTopicAdminMemberRole,
			tokenScopes: []model.Scope{model.ScopePageRead, model.ScopeTopicRead},
			topicNumber: 1,
			wantCode:    model.AppErrCodeForbidden,
		},
		{
			name:        "トークンが属さないスペースは未存在",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: writeScopes,
			identifier:  "api-page-create-auth-other",
			topicNumber: 1,
			wantCode:    model.AppErrCodeResourceNotFound,
		},
	}

	db := testutil.GetTestDB()
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupCreateAPIPageFixture(t, db, "api-page-create-auth-"+string(rune('a'+i)), tt.memberRole)
			identifier := f.space.Identifier
			if tt.identifier != "" {
				identifier = tt.identifier
			}

			output, err := newCreateAPIPageUC(db).Execute(t.Context(), CreateAPIPageInput{
				Principal:       f.principal(tt.memberRole, tt.tokenScopes),
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
					t.Errorf("エラー = %v、期待値 = AppErrorCode %d", err, tt.wantCode)
				}
			default:
				if err != nil {
					t.Fatalf("Execute()のエラー = %v", err)
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

// TestCreateAPIPageUsecase_Execute_TopicMemberScopeは、スペースのロールではpage:writeを持たないメンバーが、
// トピックメンバーとして持つpage:writeで作成できるのはそのトピックだけであることを確かめる。
// フィクスチャのメンバーはトピック2にだけトピック編集者として参加している
func TestCreateAPIPageUsecase_Execute_TopicMemberScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		tokenScopes []model.Scope
		topicNumber int32
		wantCode    model.AppErrorCode
	}{
		{
			name:        "トピック編集者として参加しているトピックには作成できる",
			tokenScopes: []model.Scope{model.ScopePageWrite, model.ScopeTopicRead},
			topicNumber: 2,
		},
		{
			name:        "参加していないトピックには作成できない",
			tokenScopes: []model.Scope{model.ScopePageWrite, model.ScopeTopicRead},
			topicNumber: 1,
			wantCode:    model.AppErrCodeForbidden,
		},
		{
			name:        "トークンがpage:writeを持たなければ、参加しているトピックにも作成できない",
			tokenScopes: []model.Scope{model.ScopePageRead, model.ScopeTopicRead},
			topicNumber: 2,
			wantCode:    model.AppErrCodeForbidden,
		},
	}

	db := testutil.GetTestDB()
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupCreateAPIPageFixture(t, db, "api-page-create-topic-member-"+string(rune('a'+i)), model.SpaceRoleViewer)

			output, err := newCreateAPIPageUC(db).Execute(t.Context(), CreateAPIPageInput{
				Principal:       f.principal(model.SpaceRoleViewer, tt.tokenScopes),
				SpaceIdentifier: f.space.Identifier,
				TopicNumber:     tt.topicNumber,
				Title:           "ページ",
				Body:            "本文",
			})

			if tt.wantCode == 0 {
				if err != nil {
					t.Fatalf("Execute()のエラー = %v", err)
				}
				if output.Page.TopicID != f.topicIDs[tt.topicNumber] {
					t.Errorf("TopicID = %s、期待値 = %s", output.Page.TopicID, f.topicIDs[tt.topicNumber])
				}
				return
			}

			if ae := model.AsAppError(err); ae == nil || ae.Code != tt.wantCode {
				t.Errorf("エラー = %v、期待値 = AppErrorCode %d", err, tt.wantCode)
			}
			if n := countRows(t, db, "pages", f.space.ID, ""); n != 0 {
				t.Errorf("ページの数 = %d、期待値 = 0", n)
			}
		})
	}
}

func TestCreateAPIPageUsecase_Execute_MaxTopicRole(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	tokenScopes := []model.Scope{model.ScopePageWrite, model.ScopeTopicRead}

	tests := []struct {
		name         string
		maxTopicRole *model.TopicRole
		wantCode     model.AppErrorCode
	}{
		{name: "上限が無ければ、閲覧者もトピック編集者として参加している非公開トピックに作成できる"},
		{name: "上限がトピック閲覧者なら、トピック編集者として参加していても作成できない", maxTopicRole: new(model.TopicRoleViewer), wantCode: model.AppErrCodeForbidden},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupCreateAPIPageFixture(t, db, "api-page-create-max-"+string(rune('a'+i)), model.SpaceRoleViewer)
			if tt.maxTopicRole != nil {
				if _, err := db.ExecContext(t.Context(),
					"UPDATE space_members SET max_topic_role = $1 WHERE id = $2 AND space_id = $3",
					string(*tt.maxTopicRole), f.spaceMemberID, f.space.ID,
				); err != nil {
					t.Fatalf("上限の設定に失敗: %v", err)
				}
			}

			// 本番の主体と同じく、スペースメンバーはRepositoryから読んだものを使う
			members, err := repository.NewSpaceMemberRepository(query.New(db)).FindByIDs(t.Context(), []model.SpaceMemberID{f.spaceMemberID}, f.space.ID)
			if err != nil || len(members) != 1 {
				t.Fatalf("スペースメンバーの読み込みに失敗: members = %v, err = %v", members, err)
			}
			principal := f.principal(model.SpaceRoleViewer, tokenScopes)
			principal.SpaceMember = members[0]

			_, err = newCreateAPIPageUC(db).Execute(t.Context(), CreateAPIPageInput{
				Principal:       principal,
				SpaceIdentifier: f.space.Identifier,
				TopicNumber:     2,
				Title:           "ページ",
				Body:            "本文",
			})
			if tt.wantCode == 0 {
				if err != nil {
					t.Fatalf("Execute()のエラー = %v", err)
				}
				return
			}
			if ae := model.AsAppError(err); ae == nil || ae.Code != tt.wantCode {
				t.Errorf("エラー = %v、期待値 = AppErrorCode %d", err, tt.wantCode)
			}
		})
	}
}

func TestCreateAPIPageUsecase_Execute_TitleConflict(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	tokenScopes := []model.Scope{model.ScopePageWrite}

	t.Run("公開済みのページと同じタイトルは422", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAPIPageFixture(t, db, "api-page-create-dup", memberRole)
		testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("重複").Build()

		output, err := newCreateAPIPageUC(db).Execute(t.Context(), CreateAPIPageInput{
			Principal:       f.principal(memberRole, tokenScopes),
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

		f := setupCreateAPIPageFixture(t, db, "api-page-create-other-topic", memberRole)
		testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[2]).WithNumber(1).WithTitle("重複").Build()

		if _, err := newCreateAPIPageUC(db).Execute(t.Context(), CreateAPIPageInput{
			Principal:       f.principal(memberRole, tokenScopes),
			SpaceIdentifier: f.space.Identifier,
			TopicNumber:     1,
			Title:           "重複",
			Body:            "本文",
		}); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
	})

	t.Run("Wikiリンクで作られた中身の無い未公開のページに本文を書いて公開する", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAPIPageFixture(t, db, "api-page-create-linked", memberRole)
		linkedID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
			WithTitle("リンク先").WithBody("").WithUnpublished().Build()

		output, err := newCreateAPIPageUC(db).Execute(t.Context(), CreateAPIPageInput{
			Principal:       f.principal(memberRole, tokenScopes),
			SpaceIdentifier: f.space.Identifier,
			TopicNumber:     1,
			Title:           "リンク先",
			Body:            "本文",
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Page.ID != linkedID || output.Page.Number != 1 {
			t.Errorf("ページ = %+v、期待値 = 番号1の既存のページ", output.Page)
		}
		if output.Page.PublishedAt == nil || output.Page.Body != "本文" {
			t.Errorf("ページ = %+v、期待値 = 本文を書いて公開したページ", output.Page)
		}
		if n := countRows(t, db, "pages", f.space.ID, ""); n != 1 {
			t.Errorf("ページの数 = %d、期待値 = 1", n)
		}
		if n := countRows(t, db, "page_revisions", f.space.ID, "page_id = $2", linkedID); n != 1 {
			t.Errorf("リビジョンの数 = %d、期待値 = 1", n)
		}
	})

	t.Run("ゴミ箱にある中身の無い未公開のページは論理削除して新しいページを作る", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAPIPageFixture(t, db, "api-page-create-trashed", memberRole)
		trashedID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
			WithTitle("リンク先").WithBody("").WithUnpublished().Build()
		if _, err := db.ExecContext(t.Context(), "UPDATE pages SET trashed_at = now() WHERE id = $1 AND space_id = $2", trashedID, f.space.ID); err != nil {
			t.Fatalf("ページをゴミ箱に入れる: %v", err)
		}

		output, err := newCreateAPIPageUC(db).Execute(t.Context(), CreateAPIPageInput{
			Principal:       f.principal(memberRole, tokenScopes),
			SpaceIdentifier: f.space.Identifier,
			TopicNumber:     1,
			Title:           "リンク先",
			Body:            "本文",
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Page.ID == trashedID || output.Page.Number != 2 || output.Page.TrashedAt != nil {
			t.Errorf("ページ = %+v、期待値 = ゴミ箱に入っていない番号2の新しいページ", output.Page)
		}
		if n := countRows(t, db, "pages", f.space.ID, "id = $2 AND discarded_at IS NOT NULL", trashedID); n != 1 {
			t.Error("ゴミ箱の未公開のページが論理削除されていない")
		}
	})
}

// TestCreateAPIPageUsecase_Execute_ReassignsLinksToReplacedPageは、ゴミ箱にある未公開のページを論理削除して
// 新しいページを作るときに、そのページを指していたリンクの記録を、新しいページへ付け替えることを確かめる
func TestCreateAPIPageUsecase_Execute_ReassignsLinksToReplacedPage(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	fixture := setupCreateAPIPageFixture(t, db, "api-page-create-reassign-link", memberRole)
	f := setupLinkReassignFixture(t, db, "api-page-create-reassign-link", fixture.space.ID, fixture.topicIDs[1], fixture.spaceMemberID, "リンク先", 1)
	if _, err := db.ExecContext(t.Context(), "UPDATE pages SET trashed_at = now() WHERE id = $1 AND space_id = $2", f.unpublishedPageID, fixture.space.ID); err != nil {
		t.Fatalf("ページをゴミ箱に入れる: %v", err)
	}

	output, err := newCreateAPIPageUC(db).Execute(t.Context(), CreateAPIPageInput{
		Principal:       fixture.principal(memberRole, []model.Scope{model.ScopePageWrite}),
		SpaceIdentifier: fixture.space.Identifier,
		TopicNumber:     1,
		Title:           "リンク先",
		Body:            "本文",
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
	if output.Page.ID == f.unpublishedPageID {
		t.Fatalf("ページ = %+v、期待値 = 新しく作ったページ", output.Page)
	}

	f.assertReassigned(t, db, output.Page.ID)
}

// TestCreateAPIPageUsecase_ExecuteWithCaseInsensitiveTitleは、検証とロック後の同名判定が
// DBのcitextの比較に揃い、大文字・小文字だけ異なる未公開ページも扱えることを確かめる。
func TestCreateAPIPageUsecase_ExecuteWithCaseInsensitiveTitle(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	for _, tt := range []struct {
		name       string
		identifier string
		trashed    bool
	}{
		{name: "ゴミ箱にない未公開ページは同じIDと番号で公開する", identifier: "api-page-case-linked"},
		{name: "ゴミ箱にある未公開ページは論理削除して新しく作る", identifier: "api-page-case-trashed", trashed: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupCreateAPIPageFixture(t, db, tt.identifier, memberRole)
			pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
				WithTitle("LinkedPage").WithBody("").WithUnpublished().Build()
			if tt.trashed {
				if _, err := db.ExecContext(t.Context(), "UPDATE pages SET trashed_at = now() WHERE id = $1 AND space_id = $2", pageID, f.space.ID); err != nil {
					t.Fatalf("ページをゴミ箱に入れる: %v", err)
				}
			}

			output, err := newCreateAPIPageUC(db).Execute(t.Context(), CreateAPIPageInput{
				Principal:       f.principal(memberRole, []model.Scope{model.ScopePageWrite}),
				SpaceIdentifier: f.space.Identifier,
				TopicNumber:     1,
				Title:           "linkedpage",
				Body:            "本文",
			})
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			if output.Page.PublishedAt == nil || output.Page.Body != "本文" || output.Page.Title == nil || *output.Page.Title != "linkedpage" || output.Page.TrashedAt != nil {
				t.Errorf("ページ = %+v、期待値 = 入力のタイトルと本文で公開したページ", output.Page)
			}
			if tt.trashed {
				if output.Page.ID == pageID || output.Page.Number != 2 {
					t.Errorf("ページ = %+v、期待値 = 番号2の新しいページ", output.Page)
				}
				if n := countRows(t, db, "pages", f.space.ID, "id = $2 AND discarded_at IS NOT NULL", pageID); n != 1 {
					t.Error("ゴミ箱の未公開ページが論理削除されていない")
				}
			} else {
				if output.Page.ID != pageID || output.Page.Number != 1 {
					t.Errorf("ページ = %+v、期待値 = 番号1の既存ページ", output.Page)
				}
				if n := countRows(t, db, "pages", f.space.ID, ""); n != 1 {
					t.Errorf("ページの数 = %d、期待値 = 1", n)
				}
			}
		})
	}
}

// TestCreateAPIPageUsecase_ExecuteForLinkedPageは、Wikiリンクで自動作成されたページのタイトルで
// 作成したときに、リンク元とリンク先が互いのリンク一覧とバックリンクに含まれ続けることと、
// リンク先にある他のスペースメンバーの下書きに触れず、公開したメンバーの最終編集日時を更新することを確かめる。
func TestCreateAPIPageUsecase_ExecuteForLinkedPage(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	f := setupCreateAPIPageFixture(t, db, "api-page-create-link-back", memberRole)
	uc := newCreateAPIPageUC(db)
	principal := f.principal(memberRole, []model.Scope{model.ScopePageWrite})

	source, err := uc.Execute(t.Context(), CreateAPIPageInput{
		Principal:       principal,
		SpaceIdentifier: f.space.Identifier,
		TopicNumber:     1,
		Title:           "リンク元",
		Body:            "[[リンク先]]",
	})
	if err != nil {
		t.Fatalf("リンク元の作成のエラー = %v", err)
	}
	if len(source.Page.LinkedPageIDs) != 1 {
		t.Fatalf("リンク元のLinkedPageIDs = %v、期待値 = 自動作成されたリンク先1件", source.Page.LinkedPageIDs)
	}
	linkedID := source.Page.LinkedPageIDs[0]
	var linkedNumber model.PageNumber
	if err := db.QueryRowContext(t.Context(), "SELECT number FROM pages WHERE id = $1 AND space_id = $2", linkedID, f.space.ID).
		Scan(&linkedNumber); err != nil {
		t.Fatalf("リンク先の番号の取得: %v", err)
	}

	pageEditorRepo := repository.NewPageEditorRepository(query.New(db))
	editorInput := repository.FindByPageAndSpaceMemberInput{
		SpaceID:       f.space.ID,
		PageID:        linkedID,
		SpaceMemberID: f.spaceMemberID,
	}
	editorBefore, err := pageEditorRepo.FindByPageAndSpaceMember(t.Context(), editorInput)
	if err != nil {
		t.Fatalf("公開前の編集者の取得: %v", err)
	}
	// 自動作成から公開までの時間の差を固定し、実行速度やDBの日時精度に依存せず更新を検証する。
	oldModifiedAt := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := pageEditorRepo.UpdateLastPageModifiedAt(t.Context(), repository.UpdateLastPageModifiedAtInput{
		ID:                 editorBefore.ID,
		SpaceID:            f.space.ID,
		LastPageModifiedAt: oldModifiedAt,
	}); err != nil {
		t.Fatalf("公開前の編集者日時の設定: %v", err)
	}

	otherUserID := testutil.NewUserBuilderDB(t, db).WithAtname("api_page_create_link_back_other").
		WithEmail("api_page_create_link_back_other@example.com").Build()
	otherMemberID := testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(f.space.ID).WithUserID(otherUserID).
		WithRole(memberRole).Build()
	testutil.NewDraftPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithPageID(linkedID).
		WithSpaceMemberID(otherMemberID).WithTitle("リンク先").WithBody("他のメンバーの下書き").Build()

	target, err := uc.Execute(t.Context(), CreateAPIPageInput{
		Principal:       principal,
		SpaceIdentifier: f.space.Identifier,
		TopicNumber:     1,
		Title:           "リンク先",
		Body:            "[[リンク元]]",
	})
	if err != nil {
		t.Fatalf("リンク先の作成のエラー = %v", err)
	}

	editorAfter, err := pageEditorRepo.FindByPageAndSpaceMember(t.Context(), editorInput)
	if err != nil {
		t.Fatalf("公開後の編集者の取得: %v", err)
	}
	if editorAfter.ID != editorBefore.ID || !editorAfter.LastPageModifiedAt.Equal(target.Page.ModifiedAt) {
		t.Errorf("編集者 = %+v、期待値 = IDを保ち最終編集日時を%vに更新", editorAfter, target.Page.ModifiedAt)
	}

	if target.Page.ID != linkedID || target.Page.Number != linkedNumber {
		t.Errorf("リンク先 = (%s, %d)、期待値 = (%s, %d)", target.Page.ID, target.Page.Number, linkedID, linkedNumber)
	}
	if len(target.Page.LinkedPageIDs) != 1 || target.Page.LinkedPageIDs[0] != source.Page.ID {
		t.Errorf("リンク先のLinkedPageIDs = %v、期待値 = [%s]", target.Page.LinkedPageIDs, source.Page.ID)
	}
	if n := countRows(t, db, "pages", f.space.ID, "id = $2 AND $3::varchar = ANY(linked_page_ids)", source.Page.ID, linkedID); n != 1 {
		t.Error("リンク元のリンク一覧にリンク先が含まれていない")
	}

	backlinks, err := repository.NewPageRepository(query.New(db)).FindBacklinkedByPageID(t.Context(), linkedID, f.space.ID)
	if err != nil {
		t.Fatalf("バックリンクの取得: %v", err)
	}
	if len(backlinks) != 1 || backlinks[0].ID != source.Page.ID {
		t.Errorf("リンク先のバックリンク = %v、期待値 = リンク元1件", backlinks)
	}

	if n := countRows(t, db, "draft_pages", f.space.ID, "page_id = $2 AND space_member_id = $3 AND body = $4", linkedID, otherMemberID, "他のメンバーの下書き"); n != 1 {
		t.Errorf("他のスペースメンバーの下書きの数 = %d、期待値 = 1", n)
	}
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
	memberRole := apiTopicRegularMemberRole
	f := setupCreateAPIPageFixture(t, db, "api-page-cc-"+uuid.NewString()[:8], memberRole)
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
				Principal:       f.principal(memberRole, []model.Scope{model.ScopePageWrite}),
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
		// wantWebPageはAPIがWeb側のページに本文を書いて公開すること
		wantWebPage bool
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
			name:        "中身の無い未公開ページとのタイトルの競合はそのページを公開する",
			webNumber:   100,
			webTitle:    "APIのページ",
			apiTitle:    "APIのページ",
			apiBody:     "本文",
			wantNumber:  100,
			wantWebPage: true,
		},
	}

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupCreateAPIPageFixture(t, db, "api-page-web-cc-"+uuid.NewString()[:8], memberRole)
			testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("既存").Build()

			webTx, webPID := beginWebTx(t, db)
			webPage := testutil.NewPageBuilder(t, webTx).
				WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(tt.webNumber).WithTitle(tt.webTitle)
			if !tt.webPublished {
				webPage = webPage.WithBody("").WithUnpublished()
			}
			webPageID := webPage.Build()

			resultCh := executeCreateAPIPageAsync(t, db, CreateAPIPageInput{
				Principal:       f.principal(memberRole, []model.Scope{model.ScopePageWrite}),
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
			if (page.ID == webPageID) != tt.wantWebPage {
				t.Errorf("APIのページのID = %s、Web側のページのID = %s、Web側のページの公開 = %t", page.ID, webPageID, tt.wantWebPage)
			}
			if n := countRows(t, db, "pages", f.space.ID, "discarded_at IS NOT NULL"); n != 0 {
				t.Errorf("論理削除されたページの数 = %d、期待値 = 0", n)
			}
		})
	}
}

// TestCreateAPIPageUsecase_ExecuteWhenReplacementIsPublishedは、APIが空の未公開ページを
// 公開すると判断した後にWeb画面が公開しても、そのページを上書きせず、タイトルの重複になることを確かめる。
func TestCreateAPIPageUsecase_ExecuteWhenReplacementIsPublished(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	f := setupCreateAPIPageFixture(t, db, "api-page-publish-cc-"+uuid.NewString()[:8], memberRole)
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
		Principal:       f.principal(memberRole, []model.Scope{model.ScopePageWrite}),
		SpaceIdentifier: f.space.Identifier,
		TopicNumber:     1,
		Title:           "同じタイトル",
		Body:            "API側の本文",
	})

	// ページ行のロックがWeb側のページ行を待ったことを確認してから公開を確定する
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

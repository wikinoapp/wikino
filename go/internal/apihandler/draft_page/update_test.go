package draft_page_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/oapi-codegen/nullable"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/apihandler/draft_page"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// newDraftPageWriteHandlerは、トークンの持ち主の下書きを持つページ1のあるスペースを作り、
// 下書きの取得・更新・削除ができるハンドラーとコンテキストを返す。
// 下書きの更新・削除はUseCaseが自前でトランザクションを張るため、フィクスチャをテストDBへコミットする
func newDraftPageWriteHandler(t *testing.T, identifier string) (*draft_page.Handler, context.Context) {
	t.Helper()

	db := testutil.GetTestDB()
	atname := "api_" + strings.ReplaceAll(identifier, "-", "_")
	userID := testutil.NewUserBuilderDB(t, db).WithAtname(atname).WithEmail(atname + "@example.com").Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier(identifier).Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(userID).Build()
	topicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).WithNumber(1).WithName("トピック").Build()
	pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(1).WithTitle("ページ").Build()
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).WithPageID(pageID).WithSpaceMemberID(spaceMemberID).WithTopicID(topicID).
		WithTitle("下書き").WithBody("下書きの本文").Build()

	ctx := middleware.SetAPIPrincipalToContext(context.Background(), &model.APIPrincipal{
		User:        &model.User{ID: userID},
		Space:       &model.Space{ID: spaceID, Identifier: model.SpaceIdentifier(identifier)},
		SpaceMember: &model.SpaceMember{ID: spaceMemberID, SpaceID: spaceID, UserID: userID, Role: model.SpaceRoleAdmin, Active: true},
		TokenKind:   model.APITokenKindPersonalAccessToken,
		Scopes:      []model.Scope{model.ScopeDraftPageRead, model.ScopeDraftPageWrite, model.ScopeDraftPageDelete},
	})

	q := query.New(db)
	pageRepo := repository.NewPageRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	h := draft_page.NewHandler(
		"https://example.com",
		usecase.NewListAPIDraftPagesUsecase(draftPageRepo, topicRepo, topicMemberRepo),
		usecase.NewGetAPIDraftPageUsecase(pageRepo, draftPageRepo, topicRepo, topicMemberRepo),
		nil,
		usecase.NewUpdateAPIDraftPageUsecase(db, repository.NewSpaceRepository(q), pageRepo, repository.NewPageEditorRepository(q),
			draftPageRepo, draftPageRevisionRepo, topicRepo, topicMemberRepo, repository.NewAttachmentRepository(q)),
		usecase.NewDeleteAPIDraftPageUsecase(db, pageRepo, draftPageRepo, draftPageRevisionRepo, topicRepo, topicMemberRepo),
	)
	return h, ctx
}

// draftPageETagは下書きを取得し、応答のETagを返す
func draftPageETag(t *testing.T, h *draft_page.Handler, ctx context.Context, identifier string) string {
	t.Helper()

	res, err := h.GetDraftPage(ctx, apigen.GetDraftPageRequestObject{SpaceIdentifier: identifier, PageNumber: 1})
	if err != nil {
		t.Fatalf("GetDraftPage()のエラー = %v", err)
	}
	ok, isOK := res.(apigen.GetDraftPage200JSONResponse)
	if !isOK {
		t.Fatalf("取得の応答の型 = %T、期待値 = GetDraftPage200JSONResponse", res)
	}
	return ok.Headers.ETag
}

func TestHandler_UpdateDraftPage(t *testing.T) {
	t.Parallel()

	const identifier = "draft-update-handler"
	h, ctx := newDraftPageWriteHandler(t, identifier)
	update := func(t *testing.T, ifMatch *string, body apigen.DraftPagePatch) apigen.UpdateDraftPage200JSONResponse {
		t.Helper()

		res, err := h.UpdateDraftPage(ctx, apigen.UpdateDraftPageRequestObject{
			SpaceIdentifier: identifier,
			PageNumber:      1,
			Params:          apigen.UpdateDraftPageParams{IfMatch: ifMatch},
			Body:            &body,
		})
		if err != nil {
			t.Fatalf("UpdateDraftPage()のエラー = %v", err)
		}
		ok, isOK := res.(apigen.UpdateDraftPage200JSONResponse)
		if !isOK {
			t.Fatalf("応答の型 = %T、期待値 = UpdateDraftPage200JSONResponse", res)
		}
		return ok
	}

	// サブテストは同じ下書きを更新するため、並列にしない。
	t.Run("If-Matchが無ければUseCaseを呼ばずにErrPreconditionRequiredを返す", func(t *testing.T) {
		_, err := h.UpdateDraftPage(ctx, apigen.UpdateDraftPageRequestObject{SpaceIdentifier: identifier, PageNumber: 1, Body: &apigen.DraftPagePatch{}})
		if !errors.Is(err, apierror.ErrPreconditionRequired) {
			t.Errorf("エラー = %v、期待値 = ErrPreconditionRequired", err)
		}
	})

	t.Run("titleを省くとタイトルを変えず、取得と同じETagを返す", func(t *testing.T) {
		etag := draftPageETag(t, h, ctx, identifier)
		body := "新しい本文"
		res := update(t, &etag, apigen.DraftPagePatch{Body: &body})
		if res.Body.Title == nil || *res.Body.Title != "下書き" || res.Body.Body != "新しい本文" {
			t.Errorf("下書き = %+v、期待値 = タイトルを変えず本文を更新した下書き", res.Body)
		}
		if res.Headers.ETag == etag || res.Headers.ETag != draftPageETag(t, h, ctx, identifier) {
			t.Errorf("ETag = %q、期待値 = 更新前 (%q) と異なり、取得と同じ値", res.Headers.ETag, etag)
		}
	})

	t.Run("titleがnullならタイトルを外す", func(t *testing.T) {
		etag := draftPageETag(t, h, ctx, identifier)
		res := update(t, &etag, apigen.DraftPagePatch{Title: nullable.NewNullNullable[string]()})
		if res.Body.Title != nil || res.Body.Body != "新しい本文" {
			t.Errorf("下書き = %+v、期待値 = タイトルを外し本文を変えない下書き", res.Body)
		}
	})

	t.Run("titleに値があればタイトルを変える", func(t *testing.T) {
		etag := draftPageETag(t, h, ctx, identifier)
		res := update(t, &etag, apigen.DraftPagePatch{Title: nullable.NewNullableWithValue("新しいタイトル")})
		if res.Body.Title == nil || *res.Body.Title != "新しいタイトル" {
			t.Errorf("タイトル = %v、期待値 = \"新しいタイトル\"", res.Body.Title)
		}
	})
}

package page_test

import (
	"context"
	"errors"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/apihandler/page"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestHandler_UpdatePage(t *testing.T) {
	t.Parallel()

	// ページの更新はUseCaseが自前でトランザクションを張るため、フィクスチャをテストDBへコミットする
	db := testutil.GetTestDB()
	userID := testutil.NewUserBuilderDB(t, db).WithAtname("api_page_update_handler").WithEmail("api_page_update_handler@example.com").Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier("page-update-handler").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(userID).Build()
	topicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).WithNumber(1).WithName("トピック1").Build()
	testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(1).WithTitle("タイトル").WithBody("本文").Build()

	ctx := middleware.SetAPIPrincipalToContext(context.Background(), &model.APIPrincipal{
		User:        &model.User{ID: userID},
		Space:       &model.Space{ID: spaceID, Identifier: "page-update-handler"},
		SpaceMember: &model.SpaceMember{ID: spaceMemberID, SpaceID: spaceID, UserID: userID, Role: model.SpaceRoleAdmin, Active: true},
		TokenKind:   model.APITokenKindPersonalAccessToken,
		Scopes:      []model.Scope{model.ScopePageRead, model.ScopePageWrite},
	})

	q := query.New(db)
	pageRepo := repository.NewPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	h := page.NewHandler(
		usecase.NewListAPIPagesUsecase(pageRepo, topicRepo, topicMemberRepo),
		usecase.NewGetAPIPageUsecase(pageRepo, topicRepo, topicMemberRepo),
		nil,
		usecase.NewUpdateAPIPageUsecase(
			db,
			repository.NewSpaceRepository(q),
			pageRepo,
			repository.NewPageRevisionRepository(q),
			repository.NewPageEditorRepository(q),
			topicRepo,
			topicMemberRepo,
			repository.NewAttachmentRepository(q),
			repository.NewPageAttachmentReferenceRepository(q),
			validator.NewAPIPageUpdateValidator(pageRepo),
		),
	)

	// getETagはページを取得し、応答のETagを返す
	getETag := func(t *testing.T) string {
		t.Helper()
		res, err := h.GetPage(ctx, apigen.GetPageRequestObject{SpaceIdentifier: "page-update-handler", PageNumber: 1})
		if err != nil {
			t.Fatalf("GetPage() error = %v", err)
		}
		fetched, ok := res.(apigen.GetPage200JSONResponse)
		if !ok {
			t.Fatalf("レスポンスの型 = %T、期待値 = GetPage200JSONResponse", res)
		}
		return fetched.Headers.ETag
	}
	update := func(ifMatch *string, body string) (apigen.UpdatePageResponseObject, error) {
		return h.UpdatePage(ctx, apigen.UpdatePageRequestObject{
			SpaceIdentifier: "page-update-handler",
			PageNumber:      1,
			Params:          apigen.UpdatePageParams{IfMatch: ifMatch},
			Body:            &apigen.UpdatePageApplicationMergePatchPlusJSONRequestBody{Body: &body},
		})
	}
	assertPreconditionFailed := func(t *testing.T, err error) {
		t.Helper()
		if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodePreconditionFailed {
			t.Errorf("error = %v、期待値 = AppErrCodePreconditionFailed", err)
		}
	}

	// サブテストは同じページを更新するため、並列にしない

	t.Run("If-Matchが無ければUseCaseを呼ばずにErrPreconditionRequiredを返す", func(t *testing.T) {
		if _, err := update(nil, "新しい本文"); !errors.Is(err, apierror.ErrPreconditionRequired) {
			t.Errorf("error = %v、期待値 = ErrPreconditionRequired", err)
		}
	})

	t.Run("弱いエンティティタグは強い比較で一致しない", func(t *testing.T) {
		weak := "W/" + getETag(t)
		_, err := update(&weak, "新しい本文")
		assertPreconditionFailed(t, err)
	})

	t.Run("引用符の無い値は一致しない", func(t *testing.T) {
		etag := getETag(t)
		unquoted := etag[1 : len(etag)-1]
		_, err := update(&unquoted, "新しい本文")
		assertPreconditionFailed(t, err)
	})

	t.Run("カンマ区切りの並びのいずれかが一致すれば更新し、取得と同じETagを返す", func(t *testing.T) {
		ifMatch := `"古い版", ` + getETag(t)
		res, err := update(&ifMatch, "新しい本文")
		if err != nil {
			t.Fatalf("UpdatePage() error = %v", err)
		}
		updated, ok := res.(apigen.UpdatePage200JSONResponse)
		if !ok {
			t.Fatalf("レスポンスの型 = %T、期待値 = UpdatePage200JSONResponse", res)
		}
		if updated.Body.Body != "新しい本文" || updated.Body.Title == nil || *updated.Body.Title != "タイトル" {
			t.Errorf("ページ = %+v、期待値 = 本文だけを更新したページ", updated.Body)
		}
		if got := getETag(t); updated.Headers.ETag != got {
			t.Errorf("更新時のETag = %q、取得時のETag = %q、期待値 = 同じ値", updated.Headers.ETag, got)
		}
	})

	t.Run("`*` はページがあれば版を問わず更新する", func(t *testing.T) {
		wildcard := " * "
		if _, err := update(&wildcard, "もう一度更新した本文"); err != nil {
			t.Fatalf("UpdatePage() error = %v", err)
		}
	})
}

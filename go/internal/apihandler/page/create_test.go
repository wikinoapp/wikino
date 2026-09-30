package page_test

import (
	"context"
	"testing"

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

func TestHandler_CreatePage(t *testing.T) {
	t.Parallel()

	// ページの作成はUseCaseが自前でトランザクションを張るため、フィクスチャをテストDBへコミットする
	db := testutil.GetTestDB()
	userID := testutil.NewUserBuilderDB(t, db).WithAtname("api_page_create_handler").WithEmail("api_page_create_handler@example.com").Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier("page-create-handler").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(userID).Build()
	testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).WithNumber(1).WithName("トピック1").Build()

	ctx := middleware.SetAPIPrincipalToContext(context.Background(), &model.APIPrincipal{
		User:        &model.User{ID: userID},
		Space:       &model.Space{ID: spaceID, Identifier: "page-create-handler"},
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
		usecase.NewCreateAPIPageUsecase(
			db,
			repository.NewSpaceRepository(q),
			pageRepo,
			repository.NewPageRevisionRepository(q),
			repository.NewPageEditorRepository(q),
			topicRepo,
			topicMemberRepo,
			repository.NewAttachmentRepository(q),
			repository.NewPageAttachmentReferenceRepository(q),
			validator.NewPageCreateValidator(pageRepo),
		),
		nil,
	)

	res, err := h.CreatePage(ctx, apigen.CreatePageRequestObject{
		SpaceIdentifier: "page-create-handler",
		Body:            &apigen.CreatePageJSONRequestBody{TopicNumber: 1, Title: "新しいページ", Body: "本文"},
	})
	if err != nil {
		t.Fatalf("CreatePage() error = %v", err)
	}
	created, ok := res.(apigen.CreatePage201JSONResponse)
	if !ok {
		t.Fatalf("レスポンスの型 = %T、期待値 = CreatePage201JSONResponse", res)
	}
	if created.Body.TopicNumber != 1 || created.Body.Title == nil || *created.Body.Title != "新しいページ" || created.Body.Body != "本文" {
		t.Errorf("ページ = %+v、期待値 = トピック1の「新しいページ」", created.Body)
	}
	if want := "/api/v1/spaces/page-create-handler/pages/1"; created.Headers.Location != want {
		t.Errorf("Location = %q、期待値 = %q", created.Headers.Location, want)
	}

	// 作成直後のETagは取得と同じ値にする
	got, err := h.GetPage(ctx, apigen.GetPageRequestObject{SpaceIdentifier: "page-create-handler", PageNumber: created.Body.Number})
	if err != nil {
		t.Fatalf("GetPage() error = %v", err)
	}
	fetched, ok := got.(apigen.GetPage200JSONResponse)
	if !ok {
		t.Fatalf("レスポンスの型 = %T、期待値 = GetPage200JSONResponse", got)
	}
	if created.Headers.ETag != fetched.Headers.ETag {
		t.Errorf("作成時のETag = %q、取得時のETag = %q、期待値 = 同じ値", created.Headers.ETag, fetched.Headers.ETag)
	}
	if !created.Body.ModifiedAt.Equal(fetched.Body.ModifiedAt) {
		t.Errorf("作成時のModifiedAt = %s、取得時 = %s、期待値 = 同じ値", created.Body.ModifiedAt, fetched.Body.ModifiedAt)
	}
}

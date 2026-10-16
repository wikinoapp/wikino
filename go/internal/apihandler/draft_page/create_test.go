package draft_page_test

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/apihandler/draft_page"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestHandler_CreateDraftPage(t *testing.T) {
	t.Parallel()

	// 下書きの作成はUseCaseが自前でトランザクションを張るため、フィクスチャをテストDBへコミットする
	db := testutil.GetTestDB()
	userID := testutil.NewUserBuilderDB(t, db).WithAtname("api_draft_create_handler").WithEmail("api_draft_create_handler@example.com").Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier("draft-create-handler").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(userID).Build()
	topicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).WithNumber(3).WithName("トピック3").Build()
	testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(1).WithTitle("ページ").WithBody("本文").Build()

	ctx := middleware.SetAPIPrincipalToContext(context.Background(), &model.APIPrincipal{
		User:        &model.User{ID: userID},
		Space:       &model.Space{ID: spaceID, Identifier: "draft-create-handler"},
		SpaceMember: &model.SpaceMember{ID: spaceMemberID, SpaceID: spaceID, UserID: userID, Role: model.SpaceRoleAdmin, Active: true},
		TokenKind:   model.APITokenKindPersonalAccessToken,
		Scopes:      []model.Scope{model.ScopeDraftPageRead, model.ScopeDraftPageWrite},
	})

	q := query.New(db)
	pageRepo := repository.NewPageRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	h := draft_page.NewHandler(
		"https://example.com",
		usecase.NewListAPIDraftPagesUsecase(draftPageRepo, topicRepo, topicMemberRepo),
		usecase.NewGetAPIDraftPageUsecase(pageRepo, draftPageRepo, topicRepo, topicMemberRepo),
		usecase.NewCreateAPIDraftPageUsecase(
			db,
			repository.NewSpaceRepository(q),
			pageRepo,
			repository.NewPageEditorRepository(q),
			draftPageRepo,
			repository.NewDraftPageRevisionRepository(q),
			topicRepo,
			topicMemberRepo,
			repository.NewAttachmentRepository(q),
			validator.NewAPIDraftPageCreateValidator(),
		),
		nil,
		nil,
	)

	pageNumber := int32(1)
	res, err := h.CreateDraftPage(ctx, apigen.CreateDraftPageRequestObject{
		SpaceIdentifier: "draft-create-handler",
		Body:            &apigen.CreateDraftPageJSONRequestBody{PageNumber: &pageNumber},
	})
	if err != nil {
		t.Fatalf("CreateDraftPage()のエラー = %v", err)
	}
	created, ok := res.(apigen.CreateDraftPage201JSONResponse)
	if !ok {
		t.Fatalf("レスポンスの型 = %T、期待値 = CreateDraftPage201JSONResponse", res)
	}
	if created.Body.PageNumber != 1 || created.Body.TopicNumber != 3 || created.Body.TopicName != "トピック3" || created.Body.Title == nil || *created.Body.Title != "ページ" || created.Body.Body != "本文" {
		t.Errorf("下書き = %+v、期待値 = トピック3のページ1の今の内容", created.Body)
	}
	if want := "/api/v1/spaces/draft-create-handler/pages/1/draft_page"; created.Headers.Location != want {
		t.Errorf("Location = %q、期待値 = %q", created.Headers.Location, want)
	}

	// 作成直後のETagは取得と同じ値にする
	got, err := h.GetDraftPage(ctx, apigen.GetDraftPageRequestObject{SpaceIdentifier: "draft-create-handler", PageNumber: 1})
	if err != nil {
		t.Fatalf("GetDraftPage()のエラー = %v", err)
	}
	fetched, ok := got.(apigen.GetDraftPage200JSONResponse)
	if !ok {
		t.Fatalf("レスポンスの型 = %T、期待値 = GetDraftPage200JSONResponse", got)
	}
	if created.Headers.ETag != fetched.Headers.ETag {
		t.Errorf("作成時のETag = %q、取得時のETag = %q、期待値 = 同じ値", created.Headers.ETag, fetched.Headers.ETag)
	}
}

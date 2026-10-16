package draft_page_publication_test

import (
	"context"
	"errors"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/apihandler/draft_page_publication"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestHandler_CreateDraftPagePublication(t *testing.T) {
	t.Parallel()

	const identifier = "draft-publish-handler"
	db := testutil.GetTestDB()
	userID := testutil.NewUserBuilderDB(t, db).WithAtname("api_draft_publish_handler").WithEmail("api_draft_publish_handler@example.com").Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier(identifier).Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(userID).Build()
	topicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).WithNumber(1).WithName("トピック").Build()
	pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(1).WithTitle("ページ").Build()
	draftID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).WithPageID(pageID).WithSpaceMemberID(spaceMemberID).WithTopicID(topicID).
		WithTitle("公開するタイトル").WithBody("公開する本文").Build()

	ctx := middleware.SetAPIPrincipalToContext(context.Background(), &model.APIPrincipal{
		User:        &model.User{ID: userID},
		Space:       &model.Space{ID: spaceID, Identifier: model.SpaceIdentifier(identifier)},
		SpaceMember: &model.SpaceMember{ID: spaceMemberID, SpaceID: spaceID, UserID: userID, Role: model.SpaceRoleAdmin, Active: true},
		TokenKind:   model.APITokenKindPersonalAccessToken,
		Scopes:      []model.Scope{model.ScopeDraftPageRead, model.ScopeDraftPageWrite, model.ScopePageRead, model.ScopePageWrite},
	})

	q := query.New(db)
	pageRepo := repository.NewPageRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	h := draft_page_publication.NewHandler("https://example.com", usecase.NewCreateAPIDraftPagePublicationUsecase(
		db,
		repository.NewSpaceRepository(q),
		pageRepo,
		repository.NewPageRevisionRepository(q),
		repository.NewPageEditorRepository(q),
		draftPageRepo,
		repository.NewDraftPageRevisionRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
		repository.NewAttachmentRepository(q),
		repository.NewPageAttachmentReferenceRepository(q),
		validator.NewAPIPageUpdateValidator(pageRepo),
	))

	// サブテストは同じ下書きを公開するため、並列にしない。
	t.Run("If-Matchが無ければUseCaseを呼ばずにErrPreconditionRequiredを返す", func(t *testing.T) {
		_, err := h.CreateDraftPagePublication(ctx, apigen.CreateDraftPagePublicationRequestObject{SpaceIdentifier: identifier, PageNumber: 1})
		if !errors.Is(err, apierror.ErrPreconditionRequired) {
			t.Errorf("エラー = %v、期待値 = ErrPreconditionRequired", err)
		}
	})

	t.Run("弱いエンティティタグでもダイジェストが違えば412", func(t *testing.T) {
		weak := `W/"古い版"`
		_, err := h.CreateDraftPagePublication(ctx, apigen.CreateDraftPagePublicationRequestObject{
			SpaceIdentifier: identifier,
			PageNumber:      1,
			Params:          apigen.CreateDraftPagePublicationParams{IfMatch: &weak},
		})
		if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodePreconditionFailed {
			t.Errorf("エラー = %v、期待値 = AppErrCodePreconditionFailed", err)
		}
	})

	// 強いETagで公開できることはcmd/wikinoのAPIのテストで確かめる
	t.Run("下書きのETagにW/が付いていても公開し、ページのURLとETagを付けて200を返す", func(t *testing.T) {
		draft, err := draftPageRepo.FindByID(ctx, draftID, spaceID)
		if err != nil || draft == nil {
			t.Fatalf("下書きの取得 = (%v, %v)、期待値 = 下書き", draft, err)
		}
		etag := `W/"` + draft.ContentDigest() + `"`

		res, err := h.CreateDraftPagePublication(ctx, apigen.CreateDraftPagePublicationRequestObject{
			SpaceIdentifier: identifier,
			PageNumber:      1,
			Params:          apigen.CreateDraftPagePublicationParams{IfMatch: &etag},
		})
		if err != nil {
			t.Fatalf("CreateDraftPagePublication()のエラー = %v", err)
		}
		ok, isOK := res.(apigen.CreateDraftPagePublication200JSONResponse)
		if !isOK {
			t.Fatalf("応答の型 = %T、期待値 = CreateDraftPagePublication200JSONResponse", res)
		}
		if ok.Body.Number != 1 || ok.Body.TopicNumber != 1 || ok.Body.TopicName != "トピック" || ok.Body.Title == nil || *ok.Body.Title != "公開するタイトル" || ok.Body.Body != "公開する本文" {
			t.Errorf("ページ = %+v、期待値 = 下書きの内容で公開したページ1", ok.Body)
		}
		if got, want := ok.Headers.ContentLocation, "/api/v1/spaces/draft-publish-handler/pages/1"; got != want {
			t.Errorf("Content-Location = %q、期待値 = %q", got, want)
		}

		page, err := pageRepo.FindBySpaceAndNumber(ctx, spaceID, 1)
		if err != nil || page == nil {
			t.Fatalf("ページの取得 = (%v, %v)、期待値 = ページ", page, err)
		}
		if want := `"` + page.ContentDigest() + `"`; ok.Headers.ETag != want {
			t.Errorf("ETag = %q、期待値 = 公開したページのETag %q", ok.Headers.ETag, want)
		}
	})
}

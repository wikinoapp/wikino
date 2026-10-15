package attachment_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/apihandler/attachment"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

func TestHandler_GetAttachment(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userID := testutil.NewUserBuilder(t, tx).Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("attachment-handler").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(spaceID).WithUserID(userID).Build()
	imageID := testutil.NewAttachmentBuilder(t, tx).
		WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).
		WithFilename("screenshot.png").WithContentType("image/png").WithByteSize(4096).WithBlobKey("attachment-handler-image").
		Build()
	// 種類が記録されていない添付ファイル
	untypedID := testutil.NewAttachmentBuilder(t, tx).
		WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).
		WithFilename("data").WithContentType("").WithBlobKey("attachment-handler-untyped").
		Build()

	h := attachment.NewHandler(usecase.NewGetAPIAttachmentUsecase(
		repository.NewAttachmentRepository(testutil.QueriesWithTx(tx)),
		storage.NewFakeObjectStorage(),
	))
	ctx := middleware.SetAPIPrincipalToContext(context.Background(), &model.APIPrincipal{
		User:        &model.User{ID: userID},
		Space:       &model.Space{ID: spaceID, Identifier: "attachment-handler"},
		SpaceMember: &model.SpaceMember{ID: spaceMemberID, SpaceID: spaceID, UserID: userID, Role: model.SpaceRoleAdmin, Active: true},
		TokenKind:   model.APITokenKindPersonalAccessToken,
		Scopes:      []model.Scope{model.ScopeAttachmentRead},
	})

	// サブテストは親のトランザクションを共有するため、並列にしない

	t.Run("添付ファイルの情報と署名付きURLを返す", func(t *testing.T) {
		res, err := h.GetAttachment(ctx, apigen.GetAttachmentRequestObject{
			SpaceIdentifier: "attachment-handler",
			AttachmentId:    uuid.MustParse(string(imageID)),
		})
		if err != nil {
			t.Fatalf("GetAttachment()のエラー = %v", err)
		}
		jsonRes, ok := res.(apigen.GetAttachment200JSONResponse)
		if !ok {
			t.Fatalf("レスポンスの型 = %T、期待値 = GetAttachment200JSONResponse", res)
		}
		got := jsonRes.Body
		if got.Id.String() != string(imageID) || got.Filename != "screenshot.png" || got.ContentType == nil || *got.ContentType != "image/png" || got.ByteSize != 4096 {
			t.Errorf("添付ファイル = %+v、期待値 = screenshot.png (image/png、4096バイト)", got)
		}
		if want := "https://storage.example.com/attachment-handler-image?expires_in=300"; got.DownloadUrl != want {
			t.Errorf("DownloadUrl = %q、期待値 = %q", got.DownloadUrl, want)
		}
		if got.DownloadUrlExpiresAt.Location().String() != "UTC" {
			t.Errorf("DownloadUrlExpiresAtのタイムゾーン = %v、期待値 = UTC", got.DownloadUrlExpiresAt.Location())
		}
	})

	t.Run("種類が記録されていない添付ファイルはcontent_typeをnullで返す", func(t *testing.T) {
		res, err := h.GetAttachment(ctx, apigen.GetAttachmentRequestObject{
			SpaceIdentifier: "attachment-handler",
			AttachmentId:    uuid.MustParse(string(untypedID)),
		})
		if err != nil {
			t.Fatalf("GetAttachment()のエラー = %v", err)
		}
		jsonRes, ok := res.(apigen.GetAttachment200JSONResponse)
		if !ok {
			t.Fatalf("レスポンスの型 = %T、期待値 = GetAttachment200JSONResponse", res)
		}
		if jsonRes.Body.ContentType != nil {
			t.Errorf("ContentType = %q、期待値 = nil", *jsonRes.Body.ContentType)
		}
	})

	t.Run("存在しない添付ファイルは未存在のエラーを返す", func(t *testing.T) {
		res, err := h.GetAttachment(ctx, apigen.GetAttachmentRequestObject{
			SpaceIdentifier: "attachment-handler",
			AttachmentId:    uuid.New(),
		})
		if res != nil {
			t.Errorf("レスポンス = %v、期待値 = nil", res)
		}
		if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("error = %v、期待値 = AppErrCodeResourceNotFound", err)
		}
	})
}

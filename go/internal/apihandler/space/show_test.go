package space_test

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/apihandler/space"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

func TestHandler_GetSpace(t *testing.T) {
	t.Parallel()

	ctx := middleware.SetAPIPrincipalToContext(context.Background(), &model.APIPrincipal{
		User: &model.User{ID: "0198f0a6-7c3e-7d1a-9b2c-3d4e5f607182", Atname: "alice"},
		Space: &model.Space{
			ID:         "0198f0a6-7c3e-7d1a-9b2c-3d4e5f607183",
			Identifier: "alice-wiki",
			Name:       "Aliceのwiki",
		},
		TokenKind: model.APITokenKindPersonalAccessToken,
	})
	h := space.NewHandler(usecase.NewGetAPISpaceUsecase())

	t.Run("トークンを束縛したスペースを返す", func(t *testing.T) {
		t.Parallel()

		res, err := h.GetSpace(ctx, apigen.GetSpaceRequestObject{SpaceIdentifier: "alice-wiki"})
		if err != nil {
			t.Fatalf("GetSpace() error = %v", err)
		}
		jsonRes, ok := res.(apigen.GetSpace200JSONResponse)
		if !ok {
			t.Fatalf("レスポンスの型 = %T、期待値 = GetSpace200JSONResponse", res)
		}

		if got := jsonRes.Body.Id.String(); got != "0198f0a6-7c3e-7d1a-9b2c-3d4e5f607183" {
			t.Errorf("id = %q", got)
		}
		if jsonRes.Body.Identifier != "alice-wiki" {
			t.Errorf("identifier = %q、期待値 = alice-wiki", jsonRes.Body.Identifier)
		}
		if jsonRes.Body.Name != "Aliceのwiki" {
			t.Errorf("name = %q", jsonRes.Body.Name)
		}
	})

	t.Run("束縛先と異なるスペースは未存在のエラーを返す", func(t *testing.T) {
		t.Parallel()

		res, err := h.GetSpace(ctx, apigen.GetSpaceRequestObject{SpaceIdentifier: "other-wiki"})
		if res != nil {
			t.Errorf("レスポンス = %v、期待値 = nil", res)
		}
		ae := model.AsAppError(err)
		if ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("error = %v、期待値 = AppErrCodeResourceNotFound", err)
		}
	})
}

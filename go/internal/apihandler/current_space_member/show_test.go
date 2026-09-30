package current_space_member_test

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/apihandler/current_space_member"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

func TestHandler_GetCurrentSpaceMember(t *testing.T) {
	t.Parallel()

	ctx := middleware.SetAPIPrincipalToContext(context.Background(), &model.APIPrincipal{
		User: &model.User{
			ID:          "0198f0a6-7c3e-7d1a-9b2c-3d4e5f607182",
			Email:       "alice@example.com",
			Atname:      "alice",
			Name:        "Alice",
			Description: "Wikinoを使っています",
		},
		Space: &model.Space{ID: "0198f0a6-7c3e-7d1a-9b2c-3d4e5f607183", Identifier: "alice-wiki"},
		SpaceMember: &model.SpaceMember{
			ID:      "0198f0a6-7c3e-7d1a-9b2c-3d4e5f607184",
			SpaceID: "0198f0a6-7c3e-7d1a-9b2c-3d4e5f607183",
			UserID:  "0198f0a6-7c3e-7d1a-9b2c-3d4e5f607182",
		},
		TokenKind: model.APITokenKindPersonalAccessToken,
	})
	h := current_space_member.NewHandler(usecase.NewGetAPICurrentSpaceMemberUsecase())

	t.Run("トークンの持ち主のメンバーを返す", func(t *testing.T) {
		t.Parallel()

		res, err := h.GetCurrentSpaceMember(ctx, apigen.GetCurrentSpaceMemberRequestObject{SpaceIdentifier: "alice-wiki"})
		if err != nil {
			t.Fatalf("GetCurrentSpaceMember() error = %v", err)
		}
		jsonRes, ok := res.(apigen.GetCurrentSpaceMember200JSONResponse)
		if !ok {
			t.Fatalf("レスポンスの型 = %T、期待値 = GetCurrentSpaceMember200JSONResponse", res)
		}

		// IDはユーザーではなくメンバーのものを返す
		if got := jsonRes.Body.Id.String(); got != "0198f0a6-7c3e-7d1a-9b2c-3d4e5f607184" {
			t.Errorf("id = %q、期待値 = メンバーのID", got)
		}
		if jsonRes.Body.Atname != "alice" {
			t.Errorf("atname = %q、期待値 = alice", jsonRes.Body.Atname)
		}
		if jsonRes.Body.Name != "Alice" {
			t.Errorf("name = %q、期待値 = Alice", jsonRes.Body.Name)
		}
		// レート制限のヘッダーはミドルウェアが付けるため、ハンドラーでは設定しない (上書きしない)
		if jsonRes.Headers.RateLimit != nil || jsonRes.Headers.RateLimitPolicy != nil {
			t.Error("ハンドラーがレート制限のヘッダーを設定している")
		}
	})

	t.Run("束縛先と異なるスペースは未存在のエラーを返す", func(t *testing.T) {
		t.Parallel()

		res, err := h.GetCurrentSpaceMember(ctx, apigen.GetCurrentSpaceMemberRequestObject{SpaceIdentifier: "other-wiki"})
		if res != nil {
			t.Errorf("レスポンス = %v、期待値 = nil", res)
		}
		ae := model.AsAppError(err)
		if ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("error = %v、期待値 = AppErrCodeResourceNotFound", err)
		}
	})
}

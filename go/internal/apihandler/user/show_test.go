package user_test

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/apihandler/user"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestHandler_GetUser(t *testing.T) {
	t.Parallel()

	t.Run("トークンの持ち主のユーザーを返す", func(t *testing.T) {
		t.Parallel()

		ctx := middleware.SetAPIPrincipalToContext(context.Background(), &model.APIPrincipal{
			User: &model.User{
				ID:          "0198f0a6-7c3e-7d1a-9b2c-3d4e5f607182",
				Email:       "alice@example.com",
				Atname:      "alice",
				Name:        "Alice",
				Description: "Wikinoを使っています",
			},
			Space:     &model.Space{ID: "0198f0a6-7c3e-7d1a-9b2c-3d4e5f607183", Identifier: "alice-wiki"},
			TokenKind: model.APITokenKindPersonalAccessToken,
		})

		res, err := user.NewHandler().GetUser(ctx, apigen.GetUserRequestObject{})
		if err != nil {
			t.Fatalf("GetUser() error = %v", err)
		}
		jsonRes, ok := res.(apigen.GetUser200JSONResponse)
		if !ok {
			t.Fatalf("レスポンスの型 = %T、期待値 = GetUser200JSONResponse", res)
		}

		if got := jsonRes.Body.Id.String(); got != "0198f0a6-7c3e-7d1a-9b2c-3d4e5f607182" {
			t.Errorf("id = %q", got)
		}
		if jsonRes.Body.Atname != "alice" {
			t.Errorf("atname = %q、期待値 = alice", jsonRes.Body.Atname)
		}
		if jsonRes.Body.Name != "Alice" {
			t.Errorf("name = %q、期待値 = Alice", jsonRes.Body.Name)
		}
		if jsonRes.Body.Description != "Wikinoを使っています" {
			t.Errorf("description = %q", jsonRes.Body.Description)
		}
		// レート制限のヘッダーはミドルウェアが付けるため、ハンドラーでは設定しない (上書きしない)
		if jsonRes.Headers.RateLimit != nil || jsonRes.Headers.RateLimitPolicy != nil {
			t.Error("ハンドラーがレート制限のヘッダーを設定している")
		}
	})

	t.Run("呼び出し主体が無ければエラーを返す", func(t *testing.T) {
		t.Parallel()

		res, err := user.NewHandler().GetUser(context.Background(), apigen.GetUserRequestObject{})
		if err == nil {
			t.Fatalf("GetUser() = %v、エラーを期待", res)
		}
	})
}

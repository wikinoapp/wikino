package oauth_protected_resource_metadata_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/oauth_protected_resource_metadata"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

func serveShow(t *testing.T, q *query.Queries, identifier string) *httptest.ResponseRecorder {
	t.Helper()

	handler := oauth_protected_resource_metadata.NewHandler(
		&config.Config{Domain: "example.com"},
		usecase.NewGetOAuthProtectedResourceMetadataUsecase(repository.NewSpaceRepository(q)),
	)
	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource/api/v1/spaces/"+identifier, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("space_identifier", identifier)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	handler.Show(rr, req)
	return rr
}

func TestShow(t *testing.T) {
	t.Parallel()

	t.Run("スペースのAPIを保護リソースとして記述する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		testutil.NewSpaceBuilder(t, tx).WithIdentifier("oprm-show").Build()

		rr := serveShow(t, q, "oprm-show")

		if rr.Code != http.StatusOK {
			t.Fatalf("ステータス = %d、期待値 = 200", rr.Code)
		}
		if got := rr.Header().Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q、期待値 = application/json", got)
		}
		if got := rr.Header().Get("Cache-Control"); got != "public, max-age=3600" {
			t.Errorf("Cache-Control = %q、期待値 = public, max-age=3600", got)
		}

		// 公開の契約のため、キーと値を固定の期待値で確かめる
		var got map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
			t.Fatalf("本文をJSONとして読めない: %v", err)
		}
		want := map[string]any{
			"resource":                 "https://example.com/api/v1/spaces/oprm-show",
			"authorization_servers":    []any{"https://example.com"},
			"scopes_supported":         []any{"topic:read", "page:read", "page:write"},
			"bearer_methods_supported": []any{"header"},
			"resource_documentation":   "https://example.com/api/reference/v1",
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("本文 = %v、期待値 = %v", got, want)
		}
	})

	t.Run("存在しないスペースは404", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)

		rr := serveShow(t, q, "oprm-missing")
		if rr.Code != http.StatusNotFound {
			t.Errorf("ステータス = %d、期待値 = 404", rr.Code)
		}
		if got := rr.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("Cache-Control = %q、期待値 = no-store", got)
		}
	})

	t.Run("削除済みのスペースは404", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		testutil.NewSpaceBuilder(t, tx).WithIdentifier("oprm-discarded").WithDiscarded().Build()

		rr := serveShow(t, q, "oprm-discarded")
		if rr.Code != http.StatusNotFound {
			t.Errorf("ステータス = %d、期待値 = 404", rr.Code)
		}
		if got := rr.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("Cache-Control = %q、期待値 = no-store", got)
		}
	})
}

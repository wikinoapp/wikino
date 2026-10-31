package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/api"
	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/apihandler"
	apiattachment "github.com/wikinoapp/wikino/go/internal/apihandler/attachment"
	apicurrentspacemember "github.com/wikinoapp/wikino/go/internal/apihandler/current_space_member"
	apicurrenttoken "github.com/wikinoapp/wikino/go/internal/apihandler/current_token"
	apidraftpage "github.com/wikinoapp/wikino/go/internal/apihandler/draft_page"
	apidraftpagebodyedit "github.com/wikinoapp/wikino/go/internal/apihandler/draft_page_body_edit"
	apidraftpagepublication "github.com/wikinoapp/wikino/go/internal/apihandler/draft_page_publication"
	apidraftpagerevision "github.com/wikinoapp/wikino/go/internal/apihandler/draft_page_revision"
	"github.com/wikinoapp/wikino/go/internal/apihandler/openapi_description"
	apipage "github.com/wikinoapp/wikino/go/internal/apihandler/page"
	apipagebacklink "github.com/wikinoapp/wikino/go/internal/apihandler/page_backlink"
	apipagebodyedit "github.com/wikinoapp/wikino/go/internal/apihandler/page_body_edit"
	apipagelink "github.com/wikinoapp/wikino/go/internal/apihandler/page_link"
	apispace "github.com/wikinoapp/wikino/go/internal/apihandler/space"
	apitopic "github.com/wikinoapp/wikino/go/internal/apihandler/topic"
	apiwikilinkpreview "github.com/wikinoapp/wikino/go/internal/apihandler/wikilink_preview"
	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/mcpserver"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// newMCPTestHandlerは、serve.goと同じく、トークンの照合を外した公開APIのハンドラーへツールの呼び出しを
// 委譲するMCPサーバーのハンドラーを組み立てる。公開APIのハンドラーは、DBを使わずに呼び出し主体から
// 答えるもの (getSpace・getCurrentToken) だけを本物にする
func newMCPTestHandler(t *testing.T, q *query.Queries) http.Handler {
	t.Helper()

	cfg := &config.Config{Domain: "example.com"}
	problems := apierror.NewWriter(cfg.AppURL())
	spec, err := apigen.GetSpec()
	if err != nil {
		t.Fatalf("GetSpec()のエラー = %v", err)
	}
	validator, err := middleware.NewAPIRequestValidator(spec, problems)
	if err != nil {
		t.Fatalf("NewAPIRequestValidator()のエラー = %v", err)
	}
	server := apihandler.NewServer(
		apiattachment.NewHandler(nil),
		apicurrentspacemember.NewHandler(usecase.NewGetAPICurrentSpaceMemberUsecase()),
		apicurrenttoken.NewHandler(usecase.NewGetAPICurrentTokenUsecase()),
		apidraftpage.NewHandler("https://example.com", nil, nil, nil, nil, nil),
		apidraftpagebodyedit.NewHandler("https://example.com", nil),
		apidraftpagepublication.NewHandler("https://example.com", nil),
		apidraftpagerevision.NewHandler(nil, nil),
		openapi_description.NewHandler(api.OpenAPIDescription),
		apipage.NewHandler(
			"https://example.com",
			usecase.NewListAPIPagesUsecase(repository.NewPageRepository(q), repository.NewTopicRepository(q), repository.NewTopicMemberRepository(q)),
			nil,
			nil,
			nil,
		),
		apipagebacklink.NewHandler("https://example.com", nil),
		apipagebodyedit.NewHandler("https://example.com", nil),
		apipagelink.NewHandler("https://example.com", nil),
		apispace.NewHandler(usecase.NewGetAPISpaceUsecase()),
		apitopic.NewHandler(nil, nil),
		apiwikilinkpreview.NewHandler("https://example.com", nil),
	)
	rateLimit := middleware.NewAPIRateLimit(ratelimit.NewLimiter(repository.NewRateLimitRepository(q)), problems, middleware.APISpaceMemberRateLimitPolicy)
	authenticator := usecase.NewAuthenticateAPITokenUsecase(
		repository.NewPersonalAccessTokenRepository(q),
		repository.NewOAuthAccessTokenRepository(q),
		repository.NewOAuthGrantRepository(q),
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewUserRepository(q),
		repository.NewFeatureFlagRepository(q),
	)

	mcpServer, err := mcpserver.NewServer(cfg, authenticator, api.OpenAPIDescription, newAuthenticatedAPIHandler(server, problems, rateLimit, validator),
		usecase.NewGetAPIAttachmentContentUsecase(repository.NewAttachmentRepository(q), storage.NewFakeObjectStorage()))
	if err != nil {
		t.Fatalf("NewServer()のエラー = %v", err)
	}
	// serve.goと同じく全体のchiのルーターの下に置く。ツールの呼び出しのコンテキストに外側のルーターの
	// ルートの情報が残っても、公開APIのルーターが正しくルートを探すことを確かめるため
	r := chi.NewRouter()
	r.Handle(model.MCPServerPath, mcpServer.Handler())
	return r
}

// setupMCPToolFixtureは、MCPサーバーのフラグを持つスペースidentifierのメンバーと、そのメンバーがOAuthアプリに
// 与えた許可に属するMCPサーバー宛てのアクセストークンを作り、トークンの値を返す
func setupMCPToolFixture(t *testing.T, tx *sql.Tx, identifier string) string {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(identifier + "@example.com").
		WithAtname(strings.ReplaceAll(identifier, "-", "_")).
		Build()
	testutil.NewFeatureFlagBuilder(t, tx).WithUserID(userID).WithName(string(model.FeatureFlagRemoteMCPServer)).Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier(identifier).WithName("MCPのスペース").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(spaceID).WithUserID(userID).WithRole(model.SpaceRoleEditor).Build()
	appID := testutil.NewOAuthApplicationBuilder(t, tx).WithSpaceID(spaceID).WithClientID(identifier + "-client").Build()
	grantID := testutil.NewOAuthGrantBuilder(t, tx).WithOAuthApplicationID(appID).WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).Build()

	token := string(auth.OAuthAccessTokenPrefix) + identifier
	testutil.NewOAuthAccessTokenBuilder(t, tx).
		WithOAuthGrantID(grantID).
		WithSpaceID(spaceID).
		WithTokenDigest(auth.DigestOpaqueToken(token)).
		WithScopes([]model.Scope{model.ScopePageRead}).
		WithAudience(model.OAuthAudienceMCPServer).
		WithExpiresAt(time.Now().Add(time.Hour)).
		Build()
	return token
}

// callMCPToolは、MCPサーバーのツールnameを引数argumentsで呼び、ツールの結果を返す
func callMCPTool(t *testing.T, handler http.Handler, token, name, arguments string) map[string]any {
	t.Helper()

	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":%q,"arguments":%s}}`, name, arguments)
	req := httptest.NewRequest(http.MethodPost, "https://example.com/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータス = %d、期待値 = %d (本文 = %s)", rr.Code, http.StatusOK, rr.Body.String())
	}
	var resp struct {
		Result map[string]any `json:"result"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil || resp.Result == nil {
		t.Fatalf("ツールの結果の解析に失敗: %v (本文 = %s)", err, rr.Body.String())
	}
	return resp.Result
}

func TestMCPServer_ツールの呼び出しを公開APIのハンドラーに委譲する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	handler := newMCPTestHandler(t, q)
	token := setupMCPToolFixture(t, tx, "mcp-tool-call")

	t.Run("トークンのスペースの公開APIの応答をstructuredContentで返す", func(t *testing.T) {
		result := callMCPTool(t, handler, token, "getSpace", `{}`)

		if result["isError"] == true {
			t.Fatalf("isError = true、成功を期待 (結果 = %v)", result)
		}
		structured, _ := result["structuredContent"].(map[string]any)
		if structured["identifier"] != "mcp-tool-call" || structured["name"] != "MCPのスペース" {
			t.Errorf("structuredContent = %v、期待値 = トークンのスペース", structured)
		}
	})

	t.Run("スペースより下の階層のoperationに呼び出し主体を渡して呼べる", func(t *testing.T) {
		result := callMCPTool(t, handler, token, "getCurrentToken", `{}`)

		if result["isError"] == true {
			t.Fatalf("isError = true、成功を期待 (結果 = %v)", result)
		}
		structured, _ := result["structuredContent"].(map[string]any)
		if structured["kind"] != "oauth_access_token" {
			t.Errorf("structuredContent = %v、期待値 = OAuthのアクセストークンの情報", structured)
		}
	})

	t.Run("公開APIのリクエストの検証で受け付けない引数はisErrorにする", func(t *testing.T) {
		result := callMCPTool(t, handler, token, "getPage", `{"page_number":0}`)

		if result["isError"] != true {
			t.Fatalf("isError = %v、エラーを期待 (結果 = %v)", result["isError"], result)
		}
		content, _ := result["content"].([]any)
		text, _ := content[0].(map[string]any)["text"].(string)
		if !strings.HasPrefix(text, "ステータス 422") || !strings.Contains(text, "page_number") {
			t.Errorf("テキスト = %q、期待値 = 422とpage_numberの問題", text)
		}
	})

	t.Run("トークンのスコープで呼べないツールは呼べない", func(t *testing.T) {
		body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"createPage","arguments":{"topic_number":1,"title":"x","body":""}}}`
		req := httptest.NewRequest(http.MethodPost, "https://example.com/mcp", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		var resp struct {
			Result map[string]any `json:"result"`
			Error  map[string]any `json:"error"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("本文のJSONの解析に失敗: %v (本文 = %s)", err, rr.Body.String())
		}
		if resp.Error == nil && resp.Result["isError"] != true {
			t.Errorf("本文 = %s、未知のツールとしてのエラーを期待", rr.Body.String())
		}
	})
}

func TestMCPServer_一覧のツールはincludeにbodyを渡したときだけ本文を返す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	handler := newMCPTestHandler(t, q)
	token := setupMCPToolFixture(t, tx, "mcp-tool-list")
	space, err := repository.NewSpaceRepository(q).FindByIdentifier(context.Background(), "mcp-tool-list")
	if err != nil || space == nil {
		t.Fatalf("スペースの取得の結果 = (%v, %v)、期待値 = フィクスチャのスペース", space, err)
	}
	topicID := testutil.NewTopicBuilder(t, tx).WithSpaceID(space.ID).WithNumber(1).Build()
	testutil.NewPageBuilder(t, tx).WithSpaceID(space.ID).WithTopicID(topicID).WithNumber(1).WithTitle("ページ").WithBody("# 見出し\n\n本文").Build()

	tests := []struct {
		name      string
		arguments string
		wantBody  any
	}{
		{name: "既定では本文を返さない", arguments: `{}`, wantBody: nil},
		{name: "includeにbodyを渡すと本文を返す", arguments: `{"include":["body"]}`, wantBody: "# 見出し\n\n本文"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := callMCPTool(t, handler, token, "listPages", tt.arguments)

			if result["isError"] == true {
				t.Fatalf("isError = true、成功を期待 (結果 = %v)", result)
			}
			structured, _ := result["structuredContent"].(map[string]any)
			items, _ := structured["items"].([]any)
			if len(items) != 1 {
				t.Fatalf("items = %v、期待値 = 1件", structured["items"])
			}
			item, _ := items[0].(map[string]any)
			if item["excerpt"] != "# 見出し 本文" || item["body"] != tt.wantBody {
				t.Errorf("項目 = %v、期待値 = 抜粋 %q・本文 %v", item, "# 見出し 本文", tt.wantBody)
			}
		})
	}
}

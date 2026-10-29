package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// testMCPTokenは、stubAuthenticatorが受け付けるMCPサーバー宛てのトークン
const testMCPToken = "wko_mcp"

// testOAuthApplicationIDは、stubAuthenticatorが返す呼び出し主体のOAuthアプリのID
const testOAuthApplicationID model.OAuthApplicationID = "oauth-application-1"

// stubAuthenticatorは、testMCPTokenをMCPサーバー宛てとして照合したときだけ受け付けるトークンの照合。
// トークンのスコープはscopesで、空ならpage:readだけを持つ
type stubAuthenticator struct {
	err    error
	scopes []model.Scope
}

func (s stubAuthenticator) Execute(_ context.Context, token string, audience model.OAuthAudience) (*model.APIPrincipal, error) {
	if s.err != nil {
		return nil, s.err
	}
	if token != testMCPToken || audience != model.OAuthAudienceMCPServer {
		return nil, nil
	}
	scopes := s.scopes
	if len(scopes) == 0 {
		scopes = []model.Scope{model.ScopePageRead}
	}
	oauthApplicationID := testOAuthApplicationID
	return &model.APIPrincipal{
		User:               &model.User{ID: "user-1"},
		Space:              &model.Space{ID: "space-1", Identifier: "my-space"},
		SpaceMember:        &model.SpaceMember{ID: "space-member-1"},
		TokenKind:          model.APITokenKindOAuthAccessToken,
		OAuthApplicationID: &oauthApplicationID,
		TokenExpiresAt:     time.Now().Add(time.Hour),
		Scopes:             scopes,
	}, nil
}

// initializeRequestは、MCPのinitializeの要求の本文
const initializeRequest = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}`

// serveMCPは、authenticatorでトークンを照合するMCPサーバーへ、methodとbodyの要求をトークンtokenを付けて送る
func serveMCP(t *testing.T, authenticator stubAuthenticator, method, body, token string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	return serveMCPWith(t, authenticator, http.NotFoundHandler(), method, body, token, header)
}

// serveMCPWithは、ツールの呼び出しをapiHandlerへ委譲するMCPサーバーへ、serveMCPと同じく要求を送る
func serveMCPWith(t *testing.T, authenticator stubAuthenticator, apiHandler http.Handler, method, body, token string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	return serveMCPServer(t, newTestServer(t, authenticator, apiHandler), method, body, token, header)
}

// newTestServerは、authenticatorでトークンを照合し、ツールの呼び出しをapiHandlerへ委譲するMCPサーバーを作る
func newTestServer(t *testing.T, authenticator stubAuthenticator, apiHandler http.Handler) *Server {
	t.Helper()
	return newTestServerWith(t, authenticator, apiHandler, stubAttachmentContents{})
}

// newTestServerWithは、添付ファイルの中身をattachmentContentsで読むnewTestServerと同じMCPサーバーを作る
func newTestServerWith(t *testing.T, authenticator stubAuthenticator, apiHandler http.Handler, attachmentContents attachmentContentReader) *Server {
	t.Helper()

	server, err := NewServer(&config.Config{Domain: "example.com", GitRev: "abc1234"}, authenticator, testOpenAPIDescription(t), apiHandler, attachmentContents)
	if err != nil {
		t.Fatalf("NewServer()のエラー = %v", err)
	}
	return server
}

// serveMCPServerは、MCPサーバーserverへ、methodとbodyの要求をトークンtokenを付けて送る
func serveMCPServer(t *testing.T, server *Server, method, body, token string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, "https://example.com/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for name, values := range header {
		req.Header[name] = values
	}
	rr := httptest.NewRecorder()
	// 本番と同じく、表示言語は全体に掛けるi18nのミドルウェアが決める
	i18n.Middleware(server.Handler()).ServeHTTP(rr, req)
	return rr
}

func TestHandler_MCPサーバー宛てのトークンでinitializeに答える(t *testing.T) {
	t.Parallel()

	rr := serveMCP(t, stubAuthenticator{}, http.MethodPost, initializeRequest, testMCPToken, nil)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータス = %d、期待値 = %d (本文 = %s)", rr.Code, http.StatusOK, rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Errorf("Content-Type = %q、期待値 = application/json (JSONの応答)", got)
	}

	var resp struct {
		Result struct {
			ServerInfo struct {
				Name    string `json:"name"`
				Title   string `json:"title"`
				Version string `json:"version"`
			} `json:"serverInfo"`
			Instructions string `json:"instructions"`
		} `json:"result"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("本文のJSONの解析に失敗: %v (本文 = %s)", err, rr.Body.String())
	}
	if info := resp.Result.ServerInfo; info.Name != "wikino" || info.Title != "Wikino" || info.Version != "abc1234" {
		t.Errorf("serverInfo = %+v、期待値 = wikino・Wikino・abc1234", info)
	}
	if resp.Result.Instructions != instructions || !strings.HasPrefix(instructions, "## Wikinoについて") {
		t.Errorf("instructions = %q、期待値 = 「Wikinoについて」の節", resp.Result.Instructions)
	}
}

func TestHandler_受け付けないトークンは401で保護リソースのメタデータを示す(t *testing.T) {
	t.Parallel()

	metadata := `resource_metadata="https://example.com/.well-known/oauth-protected-resource/mcp"`
	tests := []struct {
		name  string
		token string
		want  string
	}{
		// トークンの無い要求にはエラーを付けない (RFC 6750 §3.1)
		{name: "トークンが無い", want: "Bearer " + metadata},
		{name: "照合で受け付けられないトークン (公開API宛て・個人用アクセストークンなど)", token: "wko_public_api", want: "Bearer " + metadata + `, error="invalid_token"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rr := serveMCP(t, stubAuthenticator{}, http.MethodPost, initializeRequest, tt.token, nil)

			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusUnauthorized)
			}
			if got := rr.Header().Values("WWW-Authenticate"); len(got) != 1 || got[0] != tt.want {
				t.Errorf("WWW-Authenticate = %q、期待値 = [%q]", got, tt.want)
			}
		})
	}
}

func TestHandler_照合のシステムエラーは内部のエラーを見せずに500にする(t *testing.T) {
	t.Parallel()

	rr := serveMCP(t, stubAuthenticator{err: errors.New("DBに接続できない")}, http.MethodPost, initializeRequest, testMCPToken, nil)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusInternalServerError)
	}
	if strings.Contains(rr.Body.String(), "DB") {
		t.Errorf("本文 = %q、内部のエラーを含めないことを期待", rr.Body.String())
	}
}

func TestHandler_POST以外のメソッドはトークンを確かめずに405にする(t *testing.T) {
	t.Parallel()

	for _, method := range []string{http.MethodGet, http.MethodDelete, http.MethodPut} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			rr := serveMCP(t, stubAuthenticator{}, method, "", "", nil)

			if rr.Code != http.StatusMethodNotAllowed {
				t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusMethodNotAllowed)
			}
			if got := rr.Header().Get("Allow"); got != http.MethodPost {
				t.Errorf("Allow = %q、期待値 = POST", got)
			}
		})
	}
}

func TestHandler_ブラウザからの別オリジンの要求は403にする(t *testing.T) {
	t.Parallel()

	rr := serveMCP(t, stubAuthenticator{}, http.MethodPost, initializeRequest, testMCPToken, http.Header{
		"Sec-Fetch-Site": {"cross-site"},
		"Origin":         {"https://evil.example"},
	})

	if rr.Code != http.StatusForbidden {
		t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusForbidden)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q、CORSを有効にしないことを期待", got)
	}
}

// toolsListRequestは、MCPのtools/listの要求の本文
const toolsListRequest = `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`

func TestHandler_トークンのスコープで呼べるツールだけをtools_listに載せる(t *testing.T) {
	t.Parallel()

	rr := serveMCP(t, stubAuthenticator{}, http.MethodPost, toolsListRequest, testMCPToken, nil)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータス = %d、期待値 = %d (本文 = %s)", rr.Code, http.StatusOK, rr.Body.String())
	}
	var resp struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
			CacheScope string `json:"cacheScope"`
		} `json:"result"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("本文のJSONの解析に失敗: %v (本文 = %s)", err, rr.Body.String())
	}
	names := make([]string, len(resp.Result.Tools))
	for i, tool := range resp.Result.Tools {
		names[i] = tool.Name
	}
	// stubAuthenticatorのトークンはpage:readだけを持つ
	for _, want := range []string{"listPages", "getPage", "createWikilinkPreview", "getSpace", "getCurrentToken"} {
		if !slices.Contains(names, want) {
			t.Errorf("ツール = %v、%sを含むことを期待", names, want)
		}
	}
	for _, unwanted := range []string{"createPage", "updatePage", "listDraftPages", "listTopics", "getAttachment"} {
		if slices.Contains(names, unwanted) {
			t.Errorf("ツール = %v、トークンのスコープで呼べない%sを含まないことを期待", names, unwanted)
		}
	}
	if resp.Result.CacheScope != "private" {
		t.Errorf("cacheScope = %q、期待値 = private", resp.Result.CacheScope)
	}
}

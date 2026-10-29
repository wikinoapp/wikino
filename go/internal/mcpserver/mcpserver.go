// Package mcpserverは、MCPクライアントからWikinoのページと下書きを読み書きするためのリモートMCPサーバー
// (`POST /mcp`) を提供する。トランスポートはステートレスのStreamable HTTPで、応答はJSONで返す。
//
// 認証はMCPサーバーを宛先として発行したOAuthのアクセストークンだけで行い、公開API宛てのトークンと
// 個人用アクセストークンは受け付けない。
package mcpserver

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// instructionsは、MCPクライアントがモデルに渡すサーバーの説明 (`initialize` の `instructions`)。
// CLIのスキルの「Wikinoについて」の節の写しで、`make mcp-instructions-generate` で作る
//
//go:embed instructions.md
var instructions string

// serverNameは、MCPクライアントに示すサーバーの名前 (`serverInfo.name`)
const serverName = "wikino"

// apiTokenAuthenticatorは、宛先 (audience) を指定してトークンを照合するインターフェース。
// usecase.AuthenticateAPITokenUsecaseがこのインターフェースを満たす。
// トークンを受け付けられない場合は (nil, nil) を返す
type apiTokenAuthenticator interface {
	Execute(ctx context.Context, token string, audience model.OAuthAudience) (*model.APIPrincipal, error)
}

// principalKeyは、照合で得た呼び出し主体を `auth.TokenInfo` のExtraに入れるキー
const principalKey = "principal"

// Serverは、リモートMCPサーバーのエンドポイントのHTTPハンドラーを組み立てる
type Server struct {
	cfg           *config.Config
	authenticator apiTokenAuthenticator

	// toolsは、OpenAPI記述のスペース配下のoperationから作ったツール。要求のあいだで共有し、
	// 要求ごとにトークンのスコープで呼べるものだけを登録する
	tools []*operationTool

	// apiHandlerは、照合済みの呼び出し主体をコンテキストに持つ要求を受ける公開APIのハンドラー。
	// ツールの呼び出しを委譲する
	apiHandler http.Handler

	// attachmentContentsは、添付ファイルのツールで画像の中身を読む
	attachmentContents attachmentContentReader

	// loggerは、ツールの呼び出しを記録するロガー。ログに残す項目と残さない値をテストで確かめられるよう、
	// 既定のロガーを直接使わずに持たせる
	logger *slog.Logger
}

// NewServerはServerを生成する。openAPIDescriptionは公開APIのOpenAPI記述 (正本のYAML) で、
// operationIdをそのままツールの名前にするため、operationIdをGoの名前に書き換えた `apigen.GetSpec()` の
// 記述は使わない。apiHandlerは、公開APIのルーターからトークンの照合を外したハンドラーで、ツールの呼び出しを
// 委譲する。attachmentContentsは、添付ファイルのツールで画像の中身を読む。OpenAPI記述からツールを作れなければ
// エラーを返す
func NewServer(cfg *config.Config, authenticator apiTokenAuthenticator, openAPIDescription []byte, apiHandler http.Handler, attachmentContents attachmentContentReader) (*Server, error) {
	spec, err := openapi3.NewLoader().LoadFromData(openAPIDescription)
	if err != nil {
		return nil, fmt.Errorf("OpenAPI記述の読み込みに失敗: %w", err)
	}
	tools, err := newOperationTools(spec)
	if err != nil {
		return nil, fmt.Errorf("OpenAPI記述からのツールの作成に失敗: %w", err)
	}
	return &Server{cfg: cfg, authenticator: authenticator, tools: tools, apiHandler: apiHandler, attachmentContents: attachmentContents, logger: slog.Default()}, nil
}

// Handlerは `/mcp` のHTTPハンドラーを返す。次の順に処理する。
//
//  1. POST以外のメソッドは405にする (ステートレスのためGETのストリームとDELETEのセッション終了を持たない)
//  2. ブラウザからの別オリジンの要求を、http.CrossOriginProtectionで403にする。CORSは有効にしない
//  3. `Authorization: Bearer` のトークンをMCPサーバー宛てとして照合し、受け付けられなければ401と
//     保護リソースのメタデータを指す `WWW-Authenticate` を返す。トークンを送った要求には
//     `error="invalid_token"` (RFC 6750 §3.1) も添える
//  4. 照合で得た呼び出し主体をコンテキストに入れ、SDKのStreamable HTTPのハンドラーに渡す
func (s *Server) Handler() http.Handler {
	streamable := mcp.NewStreamableHTTPHandler(s.newMCPServer, &mcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
		Logger:       slog.Default(),
	})
	requireToken := auth.RequireBearerToken(s.verifyToken, &auth.RequireBearerTokenOptions{
		ResourceMetadataURL: model.MCPServerResourceMetadataURL(s.cfg.AppURL()),
	})

	protected := http.NewCrossOriginProtection().Handler(withInvalidTokenError(requireToken(withPrincipal(streamable))))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		protected.ServeHTTP(w, r)
	})
}

// newMCPServerは、要求ごとにMCPのサーバーを作り、トークンのスコープで呼べるツールだけを登録する。
// ツールはトークンのスコープで変わるため、サーバーを要求のあいだで共有しない
func (s *Server) newMCPServer(r *http.Request) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    serverName,
		Title:   "Wikino",
		Version: s.cfg.GitRev,
	}, &mcp.ServerOptions{
		Instructions: instructions,
		// `tools/list` などの結果はトークンごとに変わるため、共有のキャッシュに載せさせない。
		// SDKの既定は `public` である
		SetCacheable: func(_ context.Context, _ mcp.Request, c *mcp.Cacheable) {
			c.CacheScope = "private"
		},
	})

	// SDKがツールのハンドラーに渡すコンテキストは、プロトコルの版によってはHTTPの要求のコンテキストを
	// 引き継がない。呼び出し主体と表示言語 (公開APIのエラーの文言に使う) は、ここで要求から取り出して渡す
	principal := middleware.APIPrincipalFromContext(r.Context())
	if principal == nil {
		return server
	}
	locale := i18n.GetLocale(r.Context())
	for _, tool := range s.tools {
		if tool.isAvailableTo(principal.Scopes) {
			server.AddTool(tool.tool, s.callTool(tool, principal, locale))
		}
	}
	return server
}

// errVerifyTokenは、トークンの照合がシステムエラーで失敗したときにSDKへ返すエラー。SDKはエラーの
// 文面を500の応答の本文にそのまま書くため、内部のエラーを渡さずにこの文面を返す
var errVerifyToken = errors.New("internal server error")

// verifyTokenは、トークンをMCPサーバー宛てとして照合する (auth.TokenVerifier)
func (s *Server) verifyToken(ctx context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
	principal, err := s.authenticator.Execute(ctx, token, model.OAuthAudienceMCPServer)
	if err != nil {
		slog.ErrorContext(ctx, "MCPサーバーのトークンの照合に失敗しました", "error", err)
		return nil, errVerifyToken
	}
	if principal == nil {
		return nil, auth.ErrInvalidToken
	}
	return &auth.TokenInfo{
		Scopes:     model.ScopesToStrings(principal.Scopes),
		Expiration: principal.TokenExpiresAt,
		UserID:     string(principal.User.ID),
		Extra:      map[string]any{principalKey: principal},
	}, nil
}

// withPrincipalは、照合で得た呼び出し主体を、公開APIと同じコンテキストのキーに入れて次へ渡す
func withPrincipal(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.TokenInfoFromContext(r.Context()).Extra[principalKey].(*model.APIPrincipal)
		next.ServeHTTP(w, r.WithContext(middleware.SetAPIPrincipalToContext(r.Context(), principal)))
	})
}

// withInvalidTokenErrorは、トークンを送った要求への401の `WWW-Authenticate` に `error="invalid_token"`
// (RFC 6750 §3.1) を添える。SDKの `auth.RequireBearerToken` は保護リソースのメタデータのURLだけを付けるが、
// クライアントが「トークンを取り直せば通る」ことを知れるよう、トークンの無い要求と区別して伝える
func withInvalidTokenError(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(&invalidTokenErrorWriter{ResponseWriter: w}, r)
	})
}

// invalidTokenErrorWriterは、401を書く直前に `WWW-Authenticate` のBearerのチャレンジへ
// `error="invalid_token"` を足すhttp.ResponseWriter
type invalidTokenErrorWriter struct {
	http.ResponseWriter
}

// WriteHeaderは、401ならチャレンジにエラーを足してからステータスを書く
func (w *invalidTokenErrorWriter) WriteHeader(code int) {
	if code == http.StatusUnauthorized {
		header := w.Header()
		challenges := header.Values("WWW-Authenticate")
		for i, challenge := range challenges {
			if strings.HasPrefix(challenge, "Bearer ") {
				challenges[i] = challenge + `, error="invalid_token"`
			}
		}
		if len(challenges) == 0 {
			challenges = []string{`Bearer error="invalid_token"`}
		}
		header["Www-Authenticate"] = challenges
	}
	w.ResponseWriter.WriteHeader(code)
}

// Unwrapは、http.ResponseControllerが元のhttp.ResponseWriterの機能 (Flushなど) を使えるようにする
func (w *invalidTokenErrorWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

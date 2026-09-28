package main

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/apipagination"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// apiMountPathは公開Web APIのルーターをマウントするパス
const apiMountPath = "/api"

// spaceAPIResourceMetadataRouteは、スペースのAPIの保護リソースのメタデータのルート。
// model.SpaceAPIResourceMetadataURLが返すURLのパスにあたる
const spaceAPIResourceMetadataRoute = "/.well-known/oauth-protected-resource/api/v1/spaces/{space_identifier}"

// allowCandidateMethodsは405の `Allow` に載せうるメソッド
var allowCandidateMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodOptions,
}

// newAPIRouterは公開Web API (`/api` 配下) のルーターを作成する。
// ルートの不一致・メソッドの不一致・生成コードでのリクエストの読み取りの失敗・ハンドラーのエラーを
// すべてProblem Detailsで返す
func newAPIRouter(server apigen.StrictServerInterface, reference http.HandlerFunc, problems *apierror.Writer, tokenAuth *middleware.APITokenAuth, rateLimit *middleware.APIRateLimit, requestValidator *middleware.APIRequestValidator) *chi.Mux {
	r := chi.NewRouter()

	// サブルーター (`/v1`) はマウント時にこのNotFound・MethodNotAllowedを引き継ぐため、Routeより前に設定する
	r.NotFound(problems.NotFound)
	r.MethodNotAllowed(func(w http.ResponseWriter, req *http.Request) {
		problems.MethodNotAllowed(w, req, allowedMethods(r, req))
	})

	// 人間向けのAPIリファレンス。`/v1` のサブルーターの外に置き、トークン認証・レート制限・リクエスト検証を掛けない
	referencePath := strings.TrimPrefix(model.APIReferencePath, apiMountPath)
	r.Get(referencePath, reference)
	r.Head(referencePath, reference)

	r.Route("/v1", func(r chi.Router) {
		// リクエスト検証はoperationのsecurityをトークン認証で得た主体と照合するため、トークン認証の後に掛ける
		r.Use(tokenAuth.Middleware)
		// レート制限はトークン認証で得た主体のユーザー単位で数えるため、トークン認証の後に掛ける
		r.Use(rateLimit.Middleware)
		r.Use(requestValidator.Middleware)

		// リクエストはミドルウェアでスキーマに照らして検証済みのため、ここでの読み取りの失敗は
		// 検証をすり抜けた形式の問題として422で返す
		readErrorHandler := func(w http.ResponseWriter, req *http.Request, err error) {
			problems.ValidationFailed(w, req, []apierror.FieldError{{Detail: err.Error()}})
		}
		strictHandler := apigen.NewStrictHandlerWithOptions(server, []apigen.StrictMiddlewareFunc{withRequestURL}, apigen.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  readErrorHandler,
			ResponseErrorHandlerFunc: problems.WriteError,
		})
		apigen.HandlerWithOptions(strictHandler, apigen.ChiServerOptions{
			BaseRouter:       r,
			ErrorHandlerFunc: readErrorHandler,
		})
	})

	return r
}

// withRequestURLは、一覧のハンドラーが次のページの `Link` ヘッダーを組み立てられるよう、
// リクエストのURLをcontextに入れる。strict serverのハンドラーはリクエストを受け取らないため、ここで渡す
func withRequestURL(f apigen.StrictHandlerFunc, _ string) apigen.StrictHandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
		return f(apipagination.WithRequestURL(ctx, r.URL), w, r, request)
	}
}

// allowedMethodsはリクエストのパスでルーターが受け付けるメソッドを返す。
// chiは独自の405ハンドラーに受け付けるメソッドを渡さないため、ルーターに問い合わせて求める
func allowedMethods(router *chi.Mux, r *http.Request) []string {
	// chiと同じく、エスケープされたパスがあればそれでルートを探す
	path := r.URL.RawPath
	if path == "" {
		path = r.URL.Path
	}
	path = strings.TrimPrefix(path, apiMountPath)

	var methods []string
	for _, method := range allowCandidateMethods {
		if router.Match(chi.NewRouteContext(), method, path) {
			methods = append(methods, method)
		}
	}
	return methods
}

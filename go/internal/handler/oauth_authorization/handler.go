// Package oauth_authorizationは、OAuthの認可エンドポイント (同意画面と同意の送信) のHTTPハンドラーを提供します。
package oauth_authorization

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/a-h/templ"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/httperror"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	oauthauthorizationpages "github.com/wikinoapp/wikino/go/internal/templates/pages/oauth_authorization"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// HandlerはOAuthの認可エンドポイントを提供します。
type Handler struct {
	cfg                        *config.Config
	getOAuthAuthorizationNewUC *usecase.GetOAuthAuthorizationNewUsecase
	createOAuthAuthorizationUC *usecase.CreateOAuthAuthorizationUsecase
}

// NewHandlerはOAuthの認可エンドポイントのハンドラーを生成します。
func NewHandler(
	cfg *config.Config,
	getOAuthAuthorizationNewUC *usecase.GetOAuthAuthorizationNewUsecase,
	createOAuthAuthorizationUC *usecase.CreateOAuthAuthorizationUsecase,
) *Handler {
	return &Handler{
		cfg:                        cfg,
		getOAuthAuthorizationNewUC: getOAuthAuthorizationNewUC,
		createOAuthAuthorizationUC: createOAuthAuthorizationUC,
	}
}

// oauthAuthorizationParamNamesは、認可要求のパラメーターとして受け取り、同意の送信へ引き継ぐ名前
var oauthAuthorizationParamNames = []string{
	"client_id",
	"redirect_uri",
	"response_type",
	"scope",
	"state",
	"code_challenge",
	"code_challenge_method",
	"resource",
}

// parseOAuthAuthorizationParamsは、クエリかフォームの値から認可要求のパラメーターを取り出す。
// 2回以上送られたパラメーターは、どの値を採るかを決めずにユースケースが拒否できるよう名前を記録する
func parseOAuthAuthorizationParams(values url.Values) usecase.OAuthAuthorizationParams {
	var duplicated []string
	for _, name := range oauthAuthorizationParamNames {
		if len(values[name]) > 1 {
			duplicated = append(duplicated, name)
		}
	}

	return usecase.OAuthAuthorizationParams{
		ClientID:            values.Get("client_id"),
		RedirectURI:         values.Get("redirect_uri"),
		ResponseType:        values.Get("response_type"),
		Scope:               values.Get("scope"),
		State:               values.Get("state"),
		CodeChallenge:       values.Get("code_challenge"),
		CodeChallengeMethod: values.Get("code_challenge_method"),
		Resource:            values.Get("resource"),
		DuplicatedParams:    duplicated,
	}
}

// setResponseHeadersは、認可エンドポイントのすべての応答に付けるヘッダーを設定する。
//
//   - Content-Security-Policy・X-Frame-Options: 同意画面を他のサイトに埋め込ませず、
//     利用者に気付かれないまま許可のボタンを押させる攻撃 (クリックジャッキング) を防ぐ
//   - X-Robots-Tag: ログインした利用者向けの画面を検索エンジンにインデックスさせない
//   - Cache-Control: CSRFトークンを含む画面と、認可コードを載せたリダイレクトをキャッシュに残さない
func setResponseHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Security-Policy", "frame-ancestors 'none'")
	h.Set("X-Frame-Options", "DENY")
	h.Set("X-Robots-Tag", "noindex")
	h.Set("Cache-Control", "no-store")
}

// redirectToClientは、リダイレクトURIにparamsと認可サーバーの識別子 (`iss`、RFC 9207) を
// 付けてクライアントへ戻す。リダイレクトURIが持つ無関係なクエリパラメーターは残し、
// 認可レスポンスのパラメーターと同名のものは発行した値で置き換える。
func (h *Handler) redirectToClient(w http.ResponseWriter, r *http.Request, redirectURI string, params url.Values, status int) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		// リダイレクトURIは登録したものとの照合を通っており、ここで解析に失敗することは無い
		slog.ErrorContext(r.Context(), "リダイレクトURIの解析に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	params.Set("iss", h.cfg.AppURL())
	query := u.Query()
	for _, name := range []string{"code", "state", "iss", "error"} {
		query.Del(name)
	}
	for name, values := range params {
		query[name] = values
	}
	u.RawQuery = query.Encode()

	http.Redirect(w, r, u.String(), status)
}

// handleErrorは、ユースケースから返ったエラーをそれに応じたレスポンスへ変える。
// クライアントへ戻せる認可要求のエラーは、リダイレクトURIへ `error` と `state` を付けて戻す。
// クライアントIDかリダイレクトURIを確かめられないエラーは、クライアントへ戻さず画面で伝える。
func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error, redirectStatus int) {
	ctx := r.Context()

	if oe := model.AsOAuthAuthorizationError(err); oe != nil {
		if oe.IsRedirectable() {
			params := url.Values{"error": {string(oe.Code)}}
			if oe.State != "" {
				params.Set("state", oe.State)
			}
			h.redirectToClient(w, r, oe.RedirectURI, params, redirectStatus)
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		h.render(w, r, "oauth_authorization_invalid_request_title", nil, oauthauthorizationpages.InvalidRequest(oauthauthorizationpages.InvalidRequestData{
			Message: oe.UserMsg,
		}))
		return
	}

	if ae := model.AsAppError(err); ae != nil {
		switch ae.Code {
		case model.AppErrCodeResourceNotFound, model.AppErrCodeForbidden:
			httperror.NotFound(w, r)
		default:
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	slog.ErrorContext(ctx, "OAuthの認可要求の処理に失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// renderは認可エンドポイントの画面を、サインイン画面と同じシンプルなレイアウトへ描画する。
// 外部アプリから送られてくる画面で、特定のスペースの画面の中に置くものではないためである。
func (h *Handler) render(w http.ResponseWriter, r *http.Request, titleKey string, titleData map[string]any, content templ.Component) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, titleKey, titleData)

	if err := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Package oauth_device_verificationは、OAuthのデバイス認可でユーザーコードを入力し、許可か拒否を
// 選ぶ画面 (RFC 8628 §3.3) のHTTPハンドラーを提供します。
package oauth_device_verification

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/a-h/templ"

	"github.com/wikinoapp/wikino/go/internal/clientip"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/httperror"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	oauthdeviceverificationpages "github.com/wikinoapp/wikino/go/internal/templates/pages/oauth_device_verification"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// ユーザーコードの送信 (照合と、許可・拒否) に掛けるレート制限。コードは要求のたびに10分しか
// 有効でないため、窓も10分に合わせる。1回のログインでは照合と決定の2回を送るため、ユーザー単位は
// 入力を何度か誤っても足りる量にする。IPアドレス単位は、同じネットワークの何人かが同時にログイン
// しても足りる量にしつつ、多数のアカウントを使った総当たりを抑える
const (
	rateLimitWindow    = 10 * time.Minute
	rateLimitUserLimit = 10
	rateLimitIPLimit   = 30
)

// Handlerはデバイス認可のユーザーコードの入力と同意の画面を提供します。
type Handler struct {
	cfg                              *config.Config
	limiter                          *ratelimit.Limiter
	getOAuthDeviceVerificationNewUC  *usecase.GetOAuthDeviceVerificationNewUsecase
	getOAuthDeviceConsentUC          *usecase.GetOAuthDeviceConsentUsecase
	updateOAuthDeviceAuthorizationUC *usecase.UpdateOAuthDeviceAuthorizationUsecase
}

// NewHandlerはデバイス認可のユーザーコードの入力と同意の画面のハンドラーを生成します。
func NewHandler(
	cfg *config.Config,
	limiter *ratelimit.Limiter,
	getOAuthDeviceVerificationNewUC *usecase.GetOAuthDeviceVerificationNewUsecase,
	getOAuthDeviceConsentUC *usecase.GetOAuthDeviceConsentUsecase,
	updateOAuthDeviceAuthorizationUC *usecase.UpdateOAuthDeviceAuthorizationUsecase,
) *Handler {
	return &Handler{
		cfg:                              cfg,
		limiter:                          limiter,
		getOAuthDeviceVerificationNewUC:  getOAuthDeviceVerificationNewUC,
		getOAuthDeviceConsentUC:          getOAuthDeviceConsentUC,
		updateOAuthDeviceAuthorizationUC: updateOAuthDeviceAuthorizationUC,
	}
}

// setResponseHeadersは、ユーザーコードの入力と同意の画面のすべての応答に付けるヘッダーを設定する。
// 認可エンドポイントの同意画面と同じく、他のサイトへの埋め込み (クリックジャッキング) を防ぎ、
// 検索エンジンにインデックスさせず、CSRFトークンやユーザーコードを含む画面をキャッシュに残さない
func setResponseHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Security-Policy", "frame-ancestors 'none'")
	h.Set("X-Frame-Options", "DENY")
	h.Set("X-Robots-Tag", "noindex")
	h.Set("Cache-Control", "no-store")
	h.Set("Referrer-Policy", "no-referrer")
}

// checkRateLimitは、ユーザーコードの送信をユーザー単位とIPアドレス単位で数える。どちらかの上限を
// 超えていれば、制限が解けるまでの時間を返す。カウンターのチェックに失敗したときは、公開APIの
// レート制限と同じくログに残して送信を通す
func (h *Handler) checkRateLimit(r *http.Request, userID model.UserID) (time.Duration, bool) {
	ctx := r.Context()
	checks := []ratelimit.CheckInput{
		{Key: ratelimit.OAuthDeviceVerificationUserKey(userID.String()), Limit: rateLimitUserLimit, Window: rateLimitWindow},
		{Key: ratelimit.OAuthDeviceVerificationIPKey(clientip.GetTrustedClientIP(r, h.cfg.TrustedProxyCIDRs)), Limit: rateLimitIPLimit, Window: rateLimitWindow},
	}
	for _, check := range checks {
		result, err := h.limiter.Check(ctx, check)
		if err != nil {
			slog.ErrorContext(ctx, "レート制限のチェックに失敗", "error", err, "key", check.Key)
			continue
		}
		if !result.Allowed {
			slog.WarnContext(ctx, "デバイス認可のユーザーコードの入力がレート制限を超えた", "key", check.Key)
			return time.Until(result.ResetAt), false
		}
	}
	return 0, true
}

// handleErrorは、ユースケースから返ったエラーのうち、画面の再表示で扱わないものをレスポンスへ変える。
// フィーチャーフラグが無効なユーザーには、画面の存在を知らせないよう404を返す
func handleError(w http.ResponseWriter, r *http.Request, err error) {
	ctx := r.Context()

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

	slog.ErrorContext(ctx, "デバイス認可のユーザーコードの処理に失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// renderNewは、ユーザーコードの入力画面をstatusで描画する。userCodeは入力欄に入れておく値
func (h *Handler) renderNew(w http.ResponseWriter, r *http.Request, status int, userCode string, formErrors *model.ValidationError) {
	ctx := r.Context()
	content := oauthdeviceverificationpages.New(oauthdeviceverificationpages.NewData{
		CSRFToken:  middleware.GetCSRFTokenFromContext(ctx),
		UserCode:   userCode,
		FormErrors: formErrors,
	})
	w.WriteHeader(status)
	h.render(w, r, "oauth_device_verification_new_title", nil, content)
}

// renderRateLimitedは、レート制限を超えた送信に、入力画面を429と `Retry-After` で返す
func (h *Handler) renderRateLimited(w http.ResponseWriter, r *http.Request, retryAfter time.Duration, userCode string) {
	ctx := r.Context()
	seconds := int(retryAfter.Round(time.Second) / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(seconds))

	formErrors := model.NewValidationError()
	formErrors.AddGlobal(i18n.T(ctx, "validation_rate_limit_exceeded"))
	h.renderNew(w, r, http.StatusTooManyRequests, userCode, formErrors)
}

// renderは画面を、認可エンドポイントの同意画面と同じシンプルなレイアウトへ描画する
func (h *Handler) render(w http.ResponseWriter, r *http.Request, titleKey string, titleData map[string]any, content templ.Component) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, titleKey, titleData)

	if err := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

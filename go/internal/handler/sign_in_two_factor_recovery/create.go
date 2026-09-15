package sign_in_two_factor_recovery

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/clientip"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/redirect"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	twofactorpages "github.com/wikinoapp/wikino/go/internal/templates/pages/sign_in_two_factor"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Createはリカバリーコード検証処理を行います (POST /sign_in/two_factor/recovery)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// pending user cookieが期限切れのときも訪問者が求めたページを付けてサインインへ戻せる
	// よう、pending userの確認より前にフォームをパースする。
	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	backURL := r.FormValue("back")

	// ペンディングユーザーIDを確認
	pendingUserID := h.sessionMgr.GetPendingUserID(r)
	if pendingUserID == "" {
		// ペンディングユーザーIDがない場合はログインページにリダイレクト
		redirect.ToSignIn(w, r, backURL)
		return
	}

	recoveryCode := r.FormValue("recovery_code")
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// UseCaseを実行 (バリデーション + リカバリーコード消費 + セッション作成)
	output, err := h.createRecoveryCodeSessionUC.Execute(ctx, usecase.CreateRecoveryCodeSessionInput{
		UserID:       pendingUserID,
		RecoveryCode: recoveryCode,
		IPAddress:    clientip.GetClientIP(r),
		UserAgent:    r.UserAgent(),
	})
	if err != nil {
		// バリデーションエラー → フォーム再描画
		if ve := model.AsValidationError(err); ve != nil {
			h.renderRecoveryForm(w, r, ve, csrfToken, backURL)
			return
		}
		// アプリケーションエラー
		if ae := model.AsAppError(err); ae != nil {
			if ae.Code == model.AppErrCodeTwoFactorNotEnabled {
				slog.WarnContext(ctx, "2FAが有効でないユーザー", "user_id", pendingUserID)
				h.sessionMgr.DeletePendingUserCookie(w)
				redirect.ToSignIn(w, r, backURL)
				return
			}
			slog.ErrorContext(ctx, ae.LogString())
		} else {
			slog.ErrorContext(ctx, "リカバリーコード認証でエラー", "error", err, "user_id", pendingUserID)
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// ペンディングユーザーIDのCookieを削除
	h.sessionMgr.DeletePendingUserCookie(w)

	// セッションCookieを設定
	h.sessionMgr.SetSessionCookie(w, output.Token)

	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_sign_in_success"))

	// backパラメータが安全な遷移先ならそこへ送り、そうでなければホームへ送る。
	http.Redirect(w, r, redirect.GetSafeRedirectURL(backURL), http.StatusFound)
}

// renderRecoveryFormはリカバリーコードフォームをエラー付きでレンダリングします
func (h *Handler) renderRecoveryForm(w http.ResponseWriter, r *http.Request, formErrors *model.ValidationError, csrfToken string, backURL string) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "sign_in_two_factor_recovery_new_title")

	pageData := twofactorpages.RecoveryNewPageData{
		CSRFToken:  csrfToken,
		FormErrors: formErrors,
		BackURL:    backURL,
	}
	content := twofactorpages.RecoveryNew(pageData)
	w.WriteHeader(http.StatusUnprocessableEntity)
	err := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content).Render(ctx, w)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

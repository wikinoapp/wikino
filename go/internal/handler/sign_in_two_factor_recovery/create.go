package sign_in_two_factor_recovery

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	twofactorpages "github.com/wikinoapp/wikino/go/internal/templates/pages/sign_in_two_factor"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Create はリカバリーコード検証処理を行います (POST /sign_in/two_factor/recovery)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// ペンディングユーザーIDを確認
	pendingUserID := h.sessionMgr.GetPendingUserID(r)
	if pendingUserID == "" {
		// ペンディングユーザーIDがない場合はログインページにリダイレクト
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	// フォームをパース
	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	recoveryCode := r.FormValue("recovery_code")
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// リクエストのバリデーション
	req := NewCreateRequest(recoveryCode)
	if formErrors := req.Validate(ctx); formErrors != nil {
		h.renderRecoveryForm(w, r, formErrors, csrfToken)
		return
	}

	// リカバリーコードを検証
	err := h.verifyTwoFactorUC.VerifyRecoveryCode(ctx, usecase.VerifyRecoveryCodeInput{
		UserID:       pendingUserID,
		RecoveryCode: recoveryCode,
	})
	if err != nil {
		if errors.Is(err, usecase.ErrTwoFactorNotEnabled) {
			// 2FAが有効でない場合はログインページにリダイレクト
			slog.WarnContext(ctx, "2FAが有効でないユーザー", "user_id", pendingUserID)
			h.sessionMgr.DeletePendingUserCookie(w)
			http.Redirect(w, r, "/sign_in", http.StatusFound)
			return
		}
		if errors.Is(err, usecase.ErrInvalidRecoveryCode) {
			// リカバリーコードが無効
			formErrors := session.NewFormErrors()
			formErrors.AddGlobal(i18n.T(ctx, "sign_in_two_factor_recovery_invalid_code"))
			h.renderRecoveryForm(w, r, formErrors, csrfToken)
			return
		}
		slog.ErrorContext(ctx, "リカバリーコード検証でエラー", "error", err, "user_id", pendingUserID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// セッションを作成
	output, err := h.createUserSessionUC.Execute(ctx, usecase.CreateUserSessionInput{
		UserID:    pendingUserID,
		IPAddress: getIPAddress(r),
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		slog.ErrorContext(ctx, "セッション作成でエラー", "error", err, "user_id", pendingUserID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// ペンディングユーザーIDのCookieを削除
	h.sessionMgr.DeletePendingUserCookie(w)

	// セッションCookieを設定
	h.sessionMgr.SetSessionCookie(w, output.Token)

	// ホームにリダイレクト
	http.Redirect(w, r, "/", http.StatusFound)
}

// renderRecoveryForm はリカバリーコードフォームをエラー付きでレンダリングします
func (h *Handler) renderRecoveryForm(w http.ResponseWriter, r *http.Request, formErrors *session.FormErrors, csrfToken string) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "sign_in_two_factor_recovery_title")

	pageData := twofactorpages.RecoveryNewPageData{
		CSRFToken:  csrfToken,
		FormErrors: formErrors,
	}
	content := twofactorpages.RecoveryNew(pageData)
	err := layouts.Simple(ctx, meta, nil, h.cfg.GetAssetVersion(), content).Render(ctx, w)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// getIPAddress はリクエストからIPアドレスを取得します
func getIPAddress(r *http.Request) string {
	// X-Forwarded-For ヘッダーがある場合はそちらを優先
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// X-Real-IP ヘッダーがある場合はそちらを使用
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// どちらもない場合はRemoteAddrを使用
	return r.RemoteAddr
}

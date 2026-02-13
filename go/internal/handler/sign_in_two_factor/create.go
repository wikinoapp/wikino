package sign_in_two_factor

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/clientip"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	twofactorpages "github.com/wikinoapp/wikino/go/internal/templates/pages/sign_in_two_factor"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Create は2FAコード検証処理を行います (POST /sign_in/two_factor)
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

	totpCode := r.FormValue("totp_code")
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// バリデーション（形式チェック + DB検証）
	result := h.createValidator.Validate(ctx, CreateValidatorInput{
		UserID:   pendingUserID,
		TOTPCode: totpCode,
	})
	if result.FormErrors != nil && result.FormErrors.HasErrors() && result.Err == nil {
		// 形式バリデーションエラー
		h.renderTwoFactorForm(w, r, result.FormErrors, csrfToken)
		return
	}
	if result.Err != nil {
		if errors.Is(result.Err, ErrTwoFactorNotEnabled) {
			// 2FAが有効でない場合はログインページにリダイレクト
			slog.WarnContext(ctx, "2FAが有効でないユーザー", "user_id", pendingUserID)
			h.sessionMgr.DeletePendingUserCookie(w)
			http.Redirect(w, r, "/sign_in", http.StatusFound)
			return
		}
		if errors.Is(result.Err, ErrInvalidTOTPCode) {
			// TOTPコードが無効
			h.renderTwoFactorForm(w, r, result.FormErrors, csrfToken)
			return
		}
		slog.ErrorContext(ctx, "TOTP検証でエラー", "error", result.Err, "user_id", pendingUserID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// セッションを作成
	output, err := h.createUserSessionUC.Execute(ctx, usecase.CreateUserSessionInput{
		UserID:    pendingUserID,
		IPAddress: clientip.GetClientIP(r),
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

// renderTwoFactorForm は2FAフォームをエラー付きでレンダリングします
func (h *Handler) renderTwoFactorForm(w http.ResponseWriter, r *http.Request, formErrors *session.FormErrors, csrfToken string) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "sign_in_two_factor_title")

	pageData := twofactorpages.NewPageData{
		CSRFToken:  csrfToken,
		FormErrors: formErrors,
	}
	content := twofactorpages.New(pageData)
	err := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content).Render(ctx, w)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

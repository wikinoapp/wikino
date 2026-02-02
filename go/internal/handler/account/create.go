package account

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	accountpages "github.com/wikinoapp/wikino/go/internal/templates/pages/account"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Create はアカウントを作成します (POST /accounts)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// フォームをパース
	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	atname := r.FormValue("atname")
	password := r.FormValue("password")

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// セッションから email_confirmation_id を取得
	emailConfirmationID := h.sessionMgr.GetEmailConfirmationID(r)
	if emailConfirmationID == "" {
		http.Redirect(w, r, "/sign_up", http.StatusFound)
		return
	}

	// メール確認情報を取得
	emailConfirmation, err := h.emailConfirmationRepo.FindByID(ctx, emailConfirmationID)
	if err != nil {
		slog.ErrorContext(ctx, "メール確認情報の取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if emailConfirmation == nil {
		http.Redirect(w, r, "/sign_up", http.StatusFound)
		return
	}

	// メール確認が完了していない場合
	if !emailConfirmation.IsSucceeded() {
		http.Redirect(w, r, "/email_confirmation/edit", http.StatusFound)
		return
	}

	// リクエストのバリデーション
	req := NewCreateRequest(atname, password)
	if formErrors := req.Validate(ctx); formErrors != nil {
		h.renderAccountForm(w, r, formErrors, csrfToken, emailConfirmation.Email, atname)
		return
	}

	// ロケールを取得
	locale := i18n.GetLocale(ctx)
	var modelLocale model.Locale
	if locale == "ja" {
		modelLocale = model.LocaleJa
	} else {
		modelLocale = model.LocaleEn
	}

	// アカウントを作成
	output, err := h.createAccountUC.Execute(ctx, usecase.CreateAccountInput{
		EmailConfirmationID: emailConfirmationID,
		Email:               emailConfirmation.Email,
		Atname:              atname,
		Password:            password,
		Locale:              modelLocale,
		TimeZone:            "Asia/Tokyo",
	})
	if err != nil {
		// アットネーム重複エラーの場合
		if errors.Is(err, usecase.ErrAtnameAlreadyTaken) {
			formErrors := session.NewFormErrors()
			formErrors.AddField("atname", i18n.T(ctx, "validation_atname_already_taken"))
			h.renderAccountForm(w, r, formErrors, csrfToken, emailConfirmation.Email, atname)
			return
		}
		// メール確認未完了エラーの場合
		if errors.Is(err, usecase.ErrEmailNotConfirmed) {
			http.Redirect(w, r, "/email_confirmation/edit", http.StatusFound)
			return
		}
		slog.ErrorContext(ctx, "アカウント作成に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// セッションを作成
	sessionOutput, err := h.createUserSessionUC.Execute(ctx, usecase.CreateUserSessionInput{
		UserID:    output.UserID,
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		slog.ErrorContext(ctx, "セッション作成に失敗", "error", err, "user_id", output.UserID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// セッションCookieを設定
	h.sessionMgr.SetSessionCookie(w, sessionOutput.Token)

	// メール確認用のCookieを削除
	h.sessionMgr.DeleteEmailConfirmationCookie(w)

	// フラッシュメッセージを設定
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_account_created"))

	// ホームにリダイレクト
	http.Redirect(w, r, "/home", http.StatusFound)
}

// renderAccountForm はアカウント作成フォームをエラー付きでレンダリングします
func (h *Handler) renderAccountForm(w http.ResponseWriter, r *http.Request, formErrors *session.FormErrors, csrfToken string, email string, atname string) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "account_new_title")

	content := accountpages.New(accountpages.NewPageData{
		CSRFToken:  csrfToken,
		FormErrors: formErrors,
		Email:      email,
		Atname:     atname,
	})

	// バリデーションエラー時は 422 Unprocessable Entity を返す
	w.WriteHeader(http.StatusUnprocessableEntity)

	err := layouts.Simple(meta, nil, content).Render(ctx, w)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

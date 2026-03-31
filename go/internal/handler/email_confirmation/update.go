package email_confirmation

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	emailconfirmationpages "github.com/wikinoapp/wikino/go/internal/templates/pages/email_confirmation"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Update は確認コードを検証します (PATCH /email_confirmation)
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// セッションから email_confirmation_id を取得
	emailConfirmationID := h.sessionMgr.GetEmailConfirmationID(r)
	if emailConfirmationID == "" {
		http.Redirect(w, r, "/sign_up", http.StatusFound)
		return
	}

	// フォームをパース
	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")

	// UseCase を実行
	err := h.markEmailAsConfirmedUC.Execute(ctx, usecase.MarkEmailAsConfirmedInput{
		EmailConfirmationID: emailConfirmationID,
		Code:                code,
	})
	if err != nil {
		h.handleUpdateError(w, r, err, emailConfirmationID)
		return
	}

	// 確認成功：アカウント作成ページにリダイレクト
	http.Redirect(w, r, "/accounts/new", http.StatusFound)
}

func (h *Handler) handleUpdateError(w http.ResponseWriter, r *http.Request, err error, emailConfirmationID string) {
	ctx := r.Context()

	if ve := model.AsValidationError(err); ve != nil {
		h.renderEditForm(w, r, ve)
		return
	}

	if ae := model.AsAppError(err); ae != nil {
		if ae.Code == model.AppErrCodeConflict {
			// 既に確認済みの場合はアカウント作成ページにリダイレクト
			slog.WarnContext(ctx, "既に確認済み", "email_confirmation_id", emailConfirmationID)
			http.Redirect(w, r, "/accounts/new", http.StatusFound)
			return
		}
		slog.ErrorContext(ctx, ae.LogString())
	}

	slog.ErrorContext(ctx, "メール確認処理に失敗", "error", err, "email_confirmation_id", emailConfirmationID)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// renderEditForm は確認コード入力フォームをエラー付きでレンダリングします
func (h *Handler) renderEditForm(w http.ResponseWriter, r *http.Request, ve *model.ValidationError) {
	ctx := r.Context()

	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "email_confirmation_edit_title")

	content := emailconfirmationpages.Edit(emailconfirmationpages.EditPageData{
		CSRFToken:  csrfToken,
		FormErrors: ve,
	})

	// バリデーションエラー時は 422 Unprocessable Entity を返す
	w.WriteHeader(http.StatusUnprocessableEntity)

	err := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content).Render(ctx, w)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

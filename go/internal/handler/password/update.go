package password

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// Update はパスワードを更新します (PATCH /password)
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// フォームデータを解析
	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームデータの解析に失敗しました", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	token := r.FormValue("token")
	password := r.FormValue("password")
	passwordConfirmation := r.FormValue("password_confirmation")

	// バリデーション
	result := h.updateValidator.Validate(ctx, validator.PasswordUpdateValidatorInput{
		Token:                token,
		Password:             password,
		PasswordConfirmation: passwordConfirmation,
	})

	if result.FormErrors != nil && result.FormErrors.HasErrors() {
		// トークンエラーの場合はトークンを空にして再表示
		displayToken := token
		if result.Err != nil && (errors.Is(result.Err, validator.ErrTokenNotFound) ||
			errors.Is(result.Err, validator.ErrTokenUsed) ||
			errors.Is(result.Err, validator.ErrTokenExpired)) {
			displayToken = ""
		}
		h.renderEditForm(w, r, displayToken, result.FormErrors)
		return
	}

	// パスワード更新ユースケースを実行
	_, err := h.updatePasswordUsecase.Execute(ctx, usecase.UpdatePasswordResetInput{
		TokenID:     result.TokenID,
		UserID:      result.UserID,
		NewPassword: password,
	})
	if err != nil {
		slog.ErrorContext(ctx, "パスワードの更新に失敗しました", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// フラッシュメッセージを設定
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_password_updated"))

	// ログインページにリダイレクト
	http.Redirect(w, r, "/sign_in", http.StatusSeeOther)
}

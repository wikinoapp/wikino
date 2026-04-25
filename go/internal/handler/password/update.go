package password

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
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

	// UseCase を実行（バリデーション・永続化を統括）
	_, err := h.updatePasswordUsecase.Execute(ctx, usecase.UpdatePasswordResetInput{
		Token:                token,
		Password:             password,
		PasswordConfirmation: passwordConfirmation,
	})
	if err != nil {
		if ve := model.AsValidationError(err); ve != nil {
			// グローバルエラーがある場合はトークン関連のエラー → トークンをクリア
			displayToken := token
			if len(ve.Global) > 0 {
				displayToken = ""
			}
			h.renderEditForm(w, r, displayToken, ve)
			return
		}
		slog.ErrorContext(ctx, "パスワードの更新に失敗しました", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// フラッシュメッセージを設定
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_password_updated"))

	// ログインページにリダイレクト
	http.Redirect(w, r, "/sign_in", http.StatusSeeOther)
}

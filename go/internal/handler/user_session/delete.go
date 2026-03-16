package user_session

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Delete はログアウト処理を行う (DELETE /user_session)
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Cookieからセッショントークンを取得
	cookie, err := r.Cookie(session.CookieName)
	if err == nil && cookie.Value != "" {
		// UseCaseを使用してデータベースからセッションを削除
		if err := h.deleteUserSessionUC.Execute(ctx, usecase.DeleteUserSessionInput{
			Token: cookie.Value,
		}); err != nil {
			slog.ErrorContext(ctx, "セッションの削除に失敗しました", "error", err)
		}
	}

	// セッションCookieを削除
	h.sessionMgr.DeleteSessionCookie(w)

	// フラッシュメッセージを設定
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "signed_out_successfully"))

	// ルートにリダイレクト
	http.Redirect(w, r, "/", http.StatusFound)
}

package user_session

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/session"
)

// Delete はログアウト処理を行う (DELETE /user_session)
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Cookieからセッショントークンを取得
	cookie, err := r.Cookie(session.CookieName)
	if err == nil && cookie.Value != "" {
		// データベースからセッションを削除
		if err := h.userSessionRepo.DeleteByToken(ctx, cookie.Value); err != nil {
			slog.ErrorContext(ctx, "セッションの削除に失敗しました", "error", err)
		}
	}

	// セッションCookieを削除
	h.sessionMgr.DeleteSessionCookie(w)

	// フラッシュメッセージを設定
	h.flashMgr.SetNotice(w, i18n.T(ctx, "signed_out_successfully"))

	// ルートにリダイレクト
	http.Redirect(w, r, "/", http.StatusFound)
}

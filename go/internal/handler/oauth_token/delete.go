package oauth_token

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// Deleteはトークンの失効の要求を処理する (POST /oauth/revoke、RFC 7009)。クライアントに発行した
// アクセストークンかリフレッシュトークンを失効させる。
//
// 失効したトークンと無効なトークンのどちらにも、本文の無い200を返す (RFC 7009 §2.2)。
// Cookieのセッションを使わず、クライアントはCSRFトークンを送れないため、CSRFの検証から除外している。
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		writeError(w, r, model.OAuthTokenErrorInvalidRequest)
		return
	}

	params, errCode := parseOAuthTokenRevocationParams(r)
	if errCode != "" {
		writeError(w, r, errCode)
		return
	}

	if err := h.revokeOAuthTokenUC.Execute(ctx, params); err != nil {
		if oe := model.AsOAuthTokenError(err); oe != nil {
			writeError(w, r, oe.Code)
			return
		}
		slog.ErrorContext(ctx, "OAuthのトークンの失効の処理に失敗", "error", err)
		w.Header().Set("Cache-Control", "no-store")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
}

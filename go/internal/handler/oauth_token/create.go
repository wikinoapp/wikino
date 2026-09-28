package oauth_token

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// Createはトークン要求を処理し、アクセストークンとリフレッシュトークンを発行する
// (POST /oauth/token)。認可コードの交換 (`authorization_code`) とリフレッシュトークンによる
// 更新 (`refresh_token`) を受け付ける。
//
// Cookieのセッションを使わず、クライアントはCSRFトークンを送れないため、CSRFの検証から除外している。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		writeError(w, r, model.OAuthTokenErrorInvalidRequest)
		return
	}

	params, errCode := parseOAuthTokenParams(r)
	if errCode != "" {
		writeError(w, r, errCode)
		return
	}

	output, err := h.createOAuthTokenUC.Execute(ctx, params)
	if err != nil {
		if oe := model.AsOAuthTokenError(err); oe != nil {
			writeError(w, r, oe.Code)
			return
		}
		slog.ErrorContext(ctx, "OAuthのトークン要求の処理に失敗", "error", err)
		w.Header().Set("Cache-Control", "no-store")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	writeToken(w, r, output)
}

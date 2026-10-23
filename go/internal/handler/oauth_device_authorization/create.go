package oauth_device_authorization

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// Createはデバイス認可の要求を処理し、デバイスコードとユーザーコードを発行する
// (POST /oauth/device_authorization)。
//
// Cookieのセッションを使わず、クライアントはCSRFトークンを送れないため、CSRFの検証から除外している。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		writeError(w, r, model.OAuthTokenErrorInvalidRequest)
		return
	}

	params, errCode := parseDeviceAuthorizationParams(r)
	if errCode != "" {
		writeError(w, r, errCode)
		return
	}

	output, err := h.createOAuthDeviceAuthorizationUC.Execute(ctx, params)
	if err != nil {
		if oe := model.AsOAuthTokenError(err); oe != nil {
			writeError(w, r, oe.Code)
			return
		}
		slog.ErrorContext(ctx, "OAuthのデバイス認可の要求の処理に失敗", "error", err)
		w.Header().Set("Cache-Control", "no-store")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	writeDeviceAuthorization(w, r, output)
}

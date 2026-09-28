// Package oauth_grantは連携中のアプリ (OAuthアプリの許可) の画面のHTTPハンドラーを提供します。
package oauth_grant

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/httperror"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerは連携中のアプリの一覧と解除の画面を提供します。
type Handler struct {
	cfg                *config.Config
	flashMgr           *session.FlashManager
	getOAuthGrantsUC   *usecase.GetOAuthGrantsUsecase
	revokeOAuthGrantUC *usecase.RevokeOAuthGrantUsecase
}

// NewHandlerは連携中のアプリのハンドラーを生成します。
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	getOAuthGrantsUC *usecase.GetOAuthGrantsUsecase,
	revokeOAuthGrantUC *usecase.RevokeOAuthGrantUsecase,
) *Handler {
	return &Handler{
		cfg:                cfg,
		flashMgr:           flashMgr,
		getOAuthGrantsUC:   getOAuthGrantsUC,
		revokeOAuthGrantUC: revokeOAuthGrantUC,
	}
}

// handleErrorはユースケースから返ったエラーを、それに応じたレスポンスへ変える。閲覧者が
// 開けない画面は「見つからない」として答え、そのスペースで誰が何をできるかを外部から試して
// 確かめられないようにする。
func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error, logMsg string) {
	ctx := r.Context()

	if ae := model.AsAppError(err); ae != nil {
		switch ae.Code {
		case model.AppErrCodeResourceNotFound, model.AppErrCodeForbidden:
			httperror.NotFound(w, r)
		default:
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	slog.ErrorContext(ctx, logMsg, "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// Package oauth_application_client_secretはOAuthアプリのクライアントシークレットを再発行する
// HTTPハンドラーを提供します。
package oauth_application_client_secret

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/httperror"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// HandlerはOAuthアプリのクライアントシークレットの再発行を提供します。
type Handler struct {
	cfg                                      *config.Config
	regenerateOAuthApplicationClientSecretUC *usecase.RegenerateOAuthApplicationClientSecretUsecase
}

// NewHandlerはクライアントシークレットのハンドラーを生成します。
func NewHandler(
	cfg *config.Config,
	regenerateOAuthApplicationClientSecretUC *usecase.RegenerateOAuthApplicationClientSecretUsecase,
) *Handler {
	return &Handler{
		cfg:                                      cfg,
		regenerateOAuthApplicationClientSecretUC: regenerateOAuthApplicationClientSecretUC,
	}
}

// handleErrorはユースケースから返ったエラーを、それに応じたレスポンスへ変える。閲覧者が
// 操作できないアプリは「見つからない」として答え、そのスペースで誰が何をできるかを外部から
// 試して確かめられないようにする (OAuthアプリの画面と同じ)。
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

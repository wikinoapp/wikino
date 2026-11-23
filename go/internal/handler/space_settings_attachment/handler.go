// Package space_settings_attachmentはスペースの添付ファイルの管理のHTTPハンドラーを提供します。
package space_settings_attachment

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/httperror"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはスペースの添付ファイルの一覧と、添付ファイルの削除を提供します。
type Handler struct {
	cfg                           *config.Config
	flashMgr                      *session.FlashManager
	getSpaceSettingsAttachmentsUC *usecase.GetSpaceSettingsAttachmentsUsecase
	deleteAttachmentUC            *usecase.DeleteAttachmentUsecase
}

// NewHandlerはスペースの添付ファイルの管理のハンドラーを生成します
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	getSpaceSettingsAttachmentsUC *usecase.GetSpaceSettingsAttachmentsUsecase,
	deleteAttachmentUC *usecase.DeleteAttachmentUsecase,
) *Handler {
	return &Handler{
		cfg:                           cfg,
		flashMgr:                      flashMgr,
		getSpaceSettingsAttachmentsUC: getSpaceSettingsAttachmentsUC,
		deleteAttachmentUC:            deleteAttachmentUC,
	}
}

// requestTargetはURLが指すスペースの識別子とログイン中のユーザーを読み、読めたかどうかを返します。
func requestTarget(w http.ResponseWriter, r *http.Request) (model.SpaceIdentifier, *model.User, bool) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return "", nil, false
	}

	return model.SpaceIdentifier(chi.URLParam(r, "space_identifier")), user, true
}

// handleErrorはユースケースから返ったエラーを、それに応じたレスポンスへ変えます。
// 閲覧者が管理できないスペースと添付ファイルは「見つからない」として答え、そのスペースで誰が何を
// できるかや、ほかのスペースの添付ファイルがあるかを、外部から試して確かめられないようにします。
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

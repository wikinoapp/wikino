package space_settings_attachment

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Deleteはスペースの添付ファイルを1件削除します (DELETE /s/{space_identifier}/settings/attachments/{attachment_id})。
// 削除できたら、検索語とページを引き継がずに一覧へリダイレクトします。
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	spaceIdentifier, user, ok := requestTarget(w, r)
	if !ok {
		return
	}

	err := h.deleteAttachmentUC.Execute(ctx, usecase.DeleteAttachmentInput{
		SpaceIdentifier: spaceIdentifier,
		UserID:          user.ID,
		AttachmentID:    model.AttachmentID(chi.URLParam(r, "attachment_id")),
	})
	if err != nil {
		h.handleError(w, r, err, "添付ファイルの削除に失敗")
		return
	}

	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_attachment_deleted"))
	http.Redirect(w, r, string(templates.SpaceSettingsAttachmentsPath(viewmodel.NewSpaceIdentifier(spaceIdentifier))), http.StatusSeeOther)
}

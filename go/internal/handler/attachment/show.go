package attachment

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/httperror"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// ShowはGETまたはHEAD /attachments/{attachment_id} - 添付ファイルの中身の署名付きURLへ302でリダイレクトします。
// ログインしていなくても、参照するページがすべて公開トピックにある添付ファイルは見られます。
// 見られない添付ファイルと存在しない添付ファイルは、区別せずに404にします。
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var userID *model.UserID
	if user := middleware.UserFromContext(ctx); user != nil {
		userID = &user.ID
	}

	attachmentID := model.AttachmentID(chi.URLParam(r, "attachment_id"))
	output, err := h.getAttachmentUC.Execute(ctx, usecase.GetAttachmentInput{
		AttachmentID: attachmentID,
		UserID:       userID,
	})
	if err != nil {
		if ae := model.AsAppError(err); ae != nil && ae.Code == model.AppErrCodeResourceNotFound {
			httperror.NotFound(w, r)
			return
		}
		slog.ErrorContext(ctx, "添付ファイルの表示に失敗", "error", err, "attachment_id", string(attachmentID))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 閲覧者ごとの判定と短命の署名付きURLに依存するため、ブラウザにも途中のキャッシュにも保存させない。
	// 署名付きURLは資格情報の代わりになるため、ログに書かない
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, output.DownloadURL, http.StatusFound)
}

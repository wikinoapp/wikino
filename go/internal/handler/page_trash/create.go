package page_trash

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Create moves a page into the trash (POST /s/{space_identifier}/pages/{page_number}/trash).
//
// [Ja] Create はページをゴミ箱へ入れます (POST /s/{space_identifier}/pages/{page_number}/trash)。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	pageNumber, err := strconv.ParseInt(chi.URLParam(r, "page_number"), 10, 32)
	if err != nil {
		handler.NotFound(w, r)
		return
	}

	output, err := h.trashPageUC.Execute(ctx, usecase.TrashPageInput{
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pageNumber),
		UserID:          user.ID,
	})
	if err != nil {
		// "not allowed to trash" and "does not exist" both become a 404, so the response never
		// reveals a page the viewer may not open.
		//
		// [Ja] 「ゴミ箱に入れる権限が無い」と「存在しない」はどちらも 404 にし、開けないページの
		// 存在をレスポンスから読み取れないようにする。
		if ae := model.AsAppError(err); ae != nil {
			switch ae.Code {
			case model.AppErrCodeResourceNotFound, model.AppErrCodeForbidden:
				handler.NotFound(w, r)
			default:
				slog.ErrorContext(ctx, ae.LogString())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		slog.ErrorContext(ctx, "ページのゴミ箱への移動に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// The page itself is gone from the screen the request came from, so send the user back to the
	// topic it belonged to (the same destination as the Rails TrashedPages::CreateController).
	// The path is built from the stored identifier rather than the request URL, so that a request
	// with different casing still lands on the canonical address.
	//
	// [Ja] リクエスト元の画面からはページ自体が消えるため、属していたトピックへ戻す
	// (Rails 版 TrashedPages::CreateController と同じ遷移先)。パスはリクエスト URL ではなく
	// 保存済みの識別子から組み立て、大文字小文字が異なるリクエストでも正規のアドレスへ着地させる。
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_page_moved_to_trash"))
	topicPath := templates.TopicPath(viewmodel.NewSpaceIdentifier(output.Space.Identifier), output.Topic.Number)
	http.Redirect(w, r, string(topicPath), http.StatusSeeOther)
}

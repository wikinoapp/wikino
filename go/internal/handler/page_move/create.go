package page_move

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Create はページ移動を実行します (POST /s/{space_identifier}/pages/{page_number}/move)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 認証済みユーザーを取得
	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	// URLパラメータを取得
	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))
	pageNumberStr := chi.URLParam(r, "page_number")

	pageNumber, err := strconv.ParseInt(pageNumberStr, 10, 32)
	if err != nil {
		handler.NotFound(w, r)
		return
	}

	// フォームデータを取得
	destTopicNumber := r.FormValue("dest_topic")

	// UseCase を実行
	moveOutput, err := h.movePageUC.Execute(ctx, usecase.MovePageInput{
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pageNumber),
		UserID:          user.ID,
		DestTopicNumber: destTopicNumber,
	})
	if err != nil {
		h.handleCreateError(w, r, err, user, spaceIdentifier, int32(pageNumber))
		return
	}

	// フラッシュメッセージを設定してリダイレクト
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "page_move_success"))
	pagePath := fmt.Sprintf("/s/%s/pages/%d", string(spaceIdentifier), moveOutput.Page.Number)
	http.Redirect(w, r, pagePath, http.StatusSeeOther)
}

func (h *Handler) handleCreateError(w http.ResponseWriter, r *http.Request, err error, user *model.User, spaceIdentifier model.SpaceIdentifier, pageNumber int32) {
	ctx := r.Context()

	if ve := model.AsValidationError(err); ve != nil {
		// バリデーションエラー → フォーム再描画
		output, getErr := h.getPageMoveDataUC.Execute(ctx, usecase.GetPageMoveDataInput{
			SpaceIdentifier: spaceIdentifier,
			PageNumber:      pageNumber,
			UserID:          user.ID,
		})
		if getErr != nil {
			slog.ErrorContext(ctx, "フォーム再表示用データの取得に失敗", "error", getErr, "original_error", ve.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusUnprocessableEntity)
		h.renderMoveForm(w, r, user, spaceIdentifier, output, ve)
		return
	}

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

	slog.ErrorContext(ctx, "ページの移動に失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

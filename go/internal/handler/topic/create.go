package topic

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	topicpages "github.com/wikinoapp/wikino/go/internal/templates/pages/topic"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Create はトピックを作成します (POST /s/{space_identifier}/topics)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	input := usecase.CreateTopicInput{
		SpaceIdentifier: spaceIdentifier,
		UserID:          user.ID,
		Name:            r.FormValue("name"),
		Description:     r.FormValue("description"),
		Visibility:      r.FormValue("visibility"),
	}

	output, err := h.createTopicUsecase.Execute(ctx, input)
	if err != nil {
		h.handleCreateError(w, r, err, user, input)
		return
	}

	spaceIdentVM := viewmodel.NewSpaceIdentifier(output.Space.Identifier)
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_topic_created"))
	http.Redirect(w, r, string(templates.TopicPath(spaceIdentVM, output.Topic.Number)), http.StatusSeeOther)
}

// handleCreateError redraws the form with what was submitted when the input is refused, and
// otherwise answers the way the topic screens answer.
//
// [Ja] handleCreateError は入力が拒否されたときに送信された内容を戻したフォームを再描画し、
// それ以外はトピックの画面と同じ形で応答する。
func (h *Handler) handleCreateError(w http.ResponseWriter, r *http.Request, err error, user *model.User, input usecase.CreateTopicInput) {
	ctx := r.Context()

	ve := model.AsValidationError(err)
	if ve == nil {
		h.handleTopicError(w, r, err, "トピックの作成に失敗")
		return
	}

	output, getErr := h.getTopicNewUsecase.Execute(ctx, usecase.GetTopicNewInput{
		SpaceIdentifier: input.SpaceIdentifier,
		UserID:          user.ID,
	})
	if getErr != nil {
		h.handleTopicError(w, r, getErr, "フォーム再表示用データの取得に失敗")
		return
	}

	w.WriteHeader(http.StatusUnprocessableEntity)
	h.renderNewForm(w, r, user, topicpages.NewData{
		CSRFToken:   middleware.GetCSRFTokenFromContext(ctx),
		FormErrors:  ve,
		Space:       viewmodel.NewSpace(output.Space),
		Name:        input.Name,
		Description: input.Description,
		Visibility:  input.Visibility,
	})
}

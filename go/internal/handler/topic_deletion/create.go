package topic_deletion

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	topicpages "github.com/wikinoapp/wikino/go/internal/templates/pages/topic"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Createはトピックを削除し、スペースの画面へリダイレクトします
// (POST /s/{space_identifier}/topics/{topic_number}/settings/deletion)。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	spaceIdentifier, topicNumber, user, ok := requestTarget(w, r)
	if !ok {
		return
	}

	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	topicName := r.FormValue("topic_name")

	output, err := h.discardTopicUC.Execute(ctx, usecase.DiscardTopicInput{
		SpaceIdentifier: spaceIdentifier,
		TopicNumber:     topicNumber,
		UserID:          user.ID,
		TopicName:       topicName,
	})
	if err != nil {
		if ve := model.AsValidationError(err); ve != nil && output != nil {
			h.render(w, r, user, http.StatusUnprocessableEntity, topicpages.DeletionNewData{
				CSRFToken:  middleware.GetCSRFTokenFromContext(ctx),
				Space:      viewmodel.NewSpace(output.Space),
				Topic:      viewmodel.NewTopic(output.Topic),
				TopicName:  topicName,
				FormErrors: ve,
			})
			return
		}
		h.handleError(w, r, err, "トピックの削除に失敗")
		return
	}

	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_topic_deleted"))
	http.Redirect(w, r, string(templates.SpacePath(viewmodel.NewSpaceIdentifier(output.Space.Identifier))), http.StatusSeeOther)
}

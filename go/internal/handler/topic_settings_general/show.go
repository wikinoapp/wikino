package topic_settings_general

import (
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	topicpages "github.com/wikinoapp/wikino/go/internal/templates/pages/topic"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Show shows the general settings of a topic
// (GET /s/{space_identifier}/topics/{topic_number}/settings/general).
//
// [Ja] Show はトピックの一般設定を表示します
// (GET /s/{space_identifier}/topics/{topic_number}/settings/general)。
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	spaceIdentifier, topicNumber, user, ok := requestTarget(w, r)
	if !ok {
		return
	}

	output, err := h.getTopicSettingsGeneralUC.Execute(ctx, usecase.GetTopicSettingsGeneralInput{
		SpaceIdentifier: spaceIdentifier,
		TopicNumber:     topicNumber,
		UserID:          user.ID,
	})
	if err != nil {
		h.handleError(w, r, err, "トピックの一般設定のデータの取得に失敗")
		return
	}

	h.render(w, r, user, topicpages.SettingsGeneralData{
		CSRFToken: middleware.GetCSRFTokenFromContext(ctx),
		Space:     viewmodel.NewSpace(output.Space),
		Topic:     viewmodel.NewTopic(output.Topic),
		Fields: topicpages.FormFieldsData{
			Name:        output.Topic.Name,
			Description: output.Topic.Description,
			Visibility:  output.Topic.Visibility.String(),
		},
	})
}

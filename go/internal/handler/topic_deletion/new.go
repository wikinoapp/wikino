package topic_deletion

import (
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	topicpages "github.com/wikinoapp/wikino/go/internal/templates/pages/topic"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Newはトピックの削除の確認画面を表示します
// (GET /s/{space_identifier}/topics/{topic_number}/settings/deletion/new)。
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	spaceIdentifier, topicNumber, user, ok := requestTarget(w, r)
	if !ok {
		return
	}

	output, err := h.getTopicDeletionNewUC.Execute(ctx, usecase.GetTopicDeletionNewInput{
		SpaceIdentifier: spaceIdentifier,
		TopicNumber:     topicNumber,
		UserID:          user.ID,
	})
	if err != nil {
		h.handleError(w, r, err, "トピックの削除の確認画面のデータの取得に失敗")
		return
	}

	h.render(w, r, user, http.StatusOK, topicpages.DeletionNewData{
		CSRFToken: middleware.GetCSRFTokenFromContext(ctx),
		Space:     viewmodel.NewSpace(output.Space),
		Topic:     viewmodel.NewTopic(output.Topic),
	})
}

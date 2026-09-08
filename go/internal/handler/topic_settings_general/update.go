package topic_settings_general

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

// Update saves the general settings of a topic
// (PATCH /s/{space_identifier}/topics/{topic_number}/settings/general).
//
// [Ja] Update はトピックの一般設定を保存します
// (PATCH /s/{space_identifier}/topics/{topic_number}/settings/general)。
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
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

	input := usecase.UpdateTopicInput{
		SpaceIdentifier: spaceIdentifier,
		TopicNumber:     topicNumber,
		UserID:          user.ID,
		Name:            r.FormValue("name"),
		Description:     r.FormValue("description"),
		Visibility:      r.FormValue("visibility"),
	}

	output, err := h.updateTopicUC.Execute(ctx, input)
	if err != nil {
		h.handleUpdateError(w, r, err, user, input)
		return
	}

	spaceIdentVM := viewmodel.NewSpaceIdentifier(output.Space.Identifier)
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_topic_updated"))
	http.Redirect(w, r, string(templates.TopicSettingsGeneralPath(spaceIdentVM, output.Topic.Number)), http.StatusSeeOther)
}

// handleUpdateError redraws the form with what was submitted when the input is refused, and
// otherwise answers the way the topic settings answer. The topic is read again for the links and
// the metadata of the screen, since the refused submission changed nothing.
//
// [Ja] handleUpdateError は入力が拒否されたときに送信された内容を戻したフォームを再描画し、
// それ以外はトピック設定の画面と同じ形で応答します。拒否された送信は何も変えていないため、画面の
// リンクとメタ情報にはトピックを読み直したものを使います。
func (h *Handler) handleUpdateError(w http.ResponseWriter, r *http.Request, err error, user *model.User, input usecase.UpdateTopicInput) {
	ctx := r.Context()

	ve := model.AsValidationError(err)
	if ve == nil {
		h.handleError(w, r, err, "トピックの一般設定の更新に失敗")
		return
	}

	output, getErr := h.getTopicSettingsGeneralUC.Execute(ctx, usecase.GetTopicSettingsGeneralInput{
		SpaceIdentifier: input.SpaceIdentifier,
		TopicNumber:     input.TopicNumber,
		UserID:          user.ID,
	})
	if getErr != nil {
		h.handleError(w, r, getErr, "フォーム再表示用データの取得に失敗")
		return
	}

	w.WriteHeader(http.StatusUnprocessableEntity)
	h.render(w, r, user, topicpages.SettingsGeneralData{
		CSRFToken: middleware.GetCSRFTokenFromContext(ctx),
		Space:     viewmodel.NewSpace(output.Space),
		Topic:     viewmodel.NewTopic(output.Topic),
		Fields: topicpages.FormFieldsData{
			FormErrors:  ve,
			Name:        input.Name,
			Description: input.Description,
			Visibility:  input.Visibility,
		},
	})
}

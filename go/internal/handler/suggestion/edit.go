package suggestion

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	suggestionpages "github.com/wikinoapp/wikino/go/internal/templates/pages/suggestion"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Edit は編集提案編集フォームを表示します (GET /s/{space_identifier}/suggestions/{suggestion_number}/edit)
func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 認証済みユーザーを取得
	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	// URLパラメータを取得
	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))
	suggestionNumberStr := chi.URLParam(r, "suggestion_number")

	suggestionNumber, err := strconv.ParseInt(suggestionNumberStr, 10, 32)
	if err != nil {
		handler.NotFound(w, r)
		return
	}

	// UseCaseでデータを取得
	output, err := h.getSuggestionEditUsecase.Execute(ctx, usecase.GetSuggestionEditInput{
		SpaceIdentifier:  spaceIdentifier,
		SuggestionNumber: model.SuggestionNumber(suggestionNumber),
		UserID:           user.ID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "編集提案編集データの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if output == nil {
		handler.NotFound(w, r)
		return
	}

	// 認可チェック
	if !output.CanUpdateSuggestion {
		handler.NotFound(w, r)
		return
	}

	h.renderEditForm(w, r, user, output, nil, output.Suggestion.Title, output.Suggestion.Body)
}

// renderEditForm renders the suggestion edit form. Space-aware metadata and links are built from
// the persisted identifier in output, so it deliberately does not take one derived from URL
// parameters.
//
// [Ja] renderEditForm は編集提案編集フォームをレンダリングします。
// スペース識別子を含むメタ情報やリンクの組み立てには output に含まれる保存済みの値を使うため、
// URL パラメータ由来の識別子は受け取りません。
func (h *Handler) renderEditForm(
	w http.ResponseWriter,
	r *http.Request,
	user *model.User,
	output *usecase.GetSuggestionEditOutput,
	formErrors *model.ValidationError,
	title string,
	body string,
) {
	ctx := r.Context()

	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	spaceVM := viewmodel.NewSpace(output.Space)
	topicVM := viewmodel.NewTopic(output.Topic)
	suggestionVM := viewmodel.NewSuggestionForDetail(viewmodel.NewSuggestionForDetailInput{
		Suggestion: output.Suggestion,
		UserMap:    output.UserMap,
	})

	// Build links from the stored identifier, not from the URL, so that every link on the screen uses
	// the same form.
	//
	// [Ja] URL ではなく保存済みの識別子からリンクを組み立て、画面内のリンクの表記を揃える。
	spaceIdentVM := spaceVM.Identifier

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "suggestion_edit_title", map[string]any{
		"SuggestionNumber": output.Suggestion.Number,
		"TopicName":        output.Topic.Name,
		"SpaceName":        output.Space.Name,
	})
	meta.CurrentSpaceIdentifier = spaceIdentVM

	content := suggestionpages.Edit(suggestionpages.EditData{
		CSRFToken:  csrfToken,
		FormErrors: formErrors,
		Space:      spaceVM,
		Topic:      topicVM,
		Suggestion: suggestionVM,
		Title:      title,
		Body:       body,
	})

	if err := RenderLayout(ctx, w, RenderLayoutInput{
		User:             user,
		SpaceIdentifier:  output.Space.Identifier,
		CurrentPageName:  templates.PageNameSuggestionEdit,
		Meta:             meta,
		BreadcrumbHeader: DetailBreadcrumbHeaderData(ctx, spaceVM, topicVM, suggestionVM.Number, suggestionVM.Title, true),
		Content:          content,
	}); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

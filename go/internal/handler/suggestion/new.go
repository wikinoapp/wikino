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

// New は編集提案作成フォームを表示します (GET /s/{space_identifier}/topics/{topic_number}/suggestions/new)
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 認証済みユーザーを取得
	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	// URLパラメータを取得
	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))
	topicNumberStr := chi.URLParam(r, "topic_number")

	topicNumber, err := strconv.ParseInt(topicNumberStr, 10, 32)
	if err != nil {
		handler.NotFound(w, r)
		return
	}

	// UseCaseでデータを取得
	output, err := h.getSuggestionNewUsecase.Execute(ctx, usecase.GetSuggestionNewInput{
		SpaceIdentifier: spaceIdentifier,
		TopicNumber:     int32(topicNumber),
		UserID:          user.ID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "編集提案作成データの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if output == nil {
		handler.NotFound(w, r)
		return
	}

	// クエリパラメータで事前選択する下書きIDを取得
	selectedDraftIDs := r.URL.Query()["draft_page_ids"]

	h.renderNewForm(w, r, user, output, nil, "", "", selectedDraftIDs)
}

// renderNewForm renders the suggestion creation form. Space-aware metadata and links are built from
// the persisted identifier in output, so it deliberately does not take one derived from URL
// parameters.
//
// [Ja] renderNewForm は編集提案作成フォームをレンダリングします。
// スペース識別子を含むメタ情報やリンクの組み立てには output に含まれる保存済みの値を使うため、
// URL パラメータ由来の識別子は受け取りません。
func (h *Handler) renderNewForm(
	w http.ResponseWriter,
	r *http.Request,
	user *model.User,
	output *usecase.GetSuggestionNewOutput,
	formErrors *model.ValidationError,
	title string,
	body string,
	selectedDraftIDs []string,
) {
	ctx := r.Context()

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// ViewModelに変換
	spaceVM := viewmodel.NewSpace(output.Space)
	topicVM := viewmodel.NewTopic(output.Topic)
	draftPagesVM := viewmodel.NewDraftPagesForSuggestionNew(output.DraftPages)

	// Build links from the stored identifier, not from the URL, so that every link on the screen uses
	// the same form.
	//
	// [Ja] URL ではなく保存済みの識別子からリンクを組み立て、画面内のリンクの表記を揃える。
	spaceIdentVM := spaceVM.Identifier

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "suggestion_new_title", map[string]any{
		"TopicName": output.Topic.Name,
		"SpaceName": output.Space.Name,
	})
	meta.CurrentSpaceIdentifier = spaceIdentVM

	content := suggestionpages.New(suggestionpages.NewData{
		CSRFToken:        csrfToken,
		FormErrors:       formErrors,
		Space:            spaceVM,
		Topic:            topicVM,
		DraftPages:       draftPagesVM,
		Title:            title,
		Body:             body,
		SelectedDraftIDs: selectedDraftIDs,
	})

	if err := RenderLayout(ctx, w, RenderLayoutInput{
		User:             user,
		SpaceIdentifier:  output.Space.Identifier,
		CurrentPageName:  templates.PageNameSuggestionNew,
		Meta:             meta,
		BreadcrumbHeader: topicBreadcrumbHeaderData(ctx, spaceVM, topicVM, true),
		Content:          content,
	}); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

package suggestion_page

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	suggestionhandler "github.com/wikinoapp/wikino/go/internal/handler/suggestion"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	suggestionpagepages "github.com/wikinoapp/wikino/go/internal/templates/pages/suggestion_page"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// New は編集提案ページ追加フォームを表示します (GET /s/{space_identifier}/suggestions/{suggestion_number}/suggestion_pages/new)
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
	suggestionNumberStr := chi.URLParam(r, "suggestion_number")

	suggestionNumberInt, err := strconv.ParseInt(suggestionNumberStr, 10, 32)
	if err != nil {
		handler.NotFound(w, r)
		return
	}
	suggestionNumber := model.SuggestionNumber(suggestionNumberInt)

	// UseCaseでデータを取得
	output, err := h.getSuggestionPageNewUsecase.Execute(ctx, usecase.GetSuggestionPageNewInput{
		SpaceIdentifier:  spaceIdentifier,
		SuggestionNumber: suggestionNumber,
		UserID:           user.ID,
	})
	if err != nil {
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
		slog.ErrorContext(ctx, "編集提案ページ追加データの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.renderNewForm(w, r, user, output, nil)
}

// renderNewForm renders the suggestion page add form. Space-aware metadata and links are built from
// the persisted identifier in output, so it deliberately does not take one derived from URL
// parameters.
//
// [Ja] renderNewForm は編集提案ページ追加フォームをレンダリングします。
// スペース識別子を含むメタ情報やリンクの組み立てには output に含まれる保存済みの値を使うため、
// URL パラメータ由来の識別子は受け取りません。
func (h *Handler) renderNewForm(
	w http.ResponseWriter,
	r *http.Request,
	user *model.User,
	output *usecase.GetSuggestionPageNewOutput,
	formErrors *model.ValidationError,
) {
	ctx := r.Context()

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// ViewModelに変換
	spaceVM := viewmodel.NewSpace(output.Space)
	topicVM := viewmodel.NewTopic(output.Topic)
	suggestionVM := viewmodel.NewSuggestionForDetail(viewmodel.NewSuggestionForDetailInput{
		Suggestion: output.Suggestion,
	})
	draftPagesVM := viewmodel.NewDraftPagesForSuggestionNew(output.DraftPages)

	// Build links from the stored identifier, not from the URL, so that every link on the screen uses
	// the same form.
	//
	// [Ja] URL ではなく保存済みの識別子からリンクを組み立て、画面内のリンクの表記を揃える。
	spaceIdentVM := spaceVM.Identifier

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "suggestion_page_new_title", map[string]any{
		"SpaceName": output.Space.Name,
	})
	meta.CurrentSpaceIdentifier = spaceIdentVM

	content := suggestionpagepages.New(suggestionpagepages.NewData{
		CSRFToken:  csrfToken,
		FormErrors: formErrors,
		Space:      spaceVM,
		Topic:      topicVM,
		Suggestion: suggestionVM,
		DraftPages: draftPagesVM,
	})

	if err := suggestionhandler.RenderLayout(ctx, w, suggestionhandler.RenderLayoutInput{
		User:             user,
		SpaceIdentifier:  output.Space.Identifier,
		CurrentPageName:  templates.PageNameSuggestionPageNew,
		Meta:             meta,
		BreadcrumbHeader: suggestionhandler.DetailBreadcrumbHeaderData(ctx, spaceVM, topicVM, suggestionVM.Number, suggestionVM.Title, true),
		Content:          content,
	}); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

package suggestion

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
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

	h.renderNewForm(w, r, user, spaceIdentifier, output, nil, "", "", selectedDraftIDs)
}

// renderNewForm は編集提案作成フォームをレンダリングします
func (h *Handler) renderNewForm(
	w http.ResponseWriter,
	r *http.Request,
	user *model.User,
	spaceIdentifier model.SpaceIdentifier,
	output *usecase.GetSuggestionNewOutput,
	formErrors *session.FormErrors,
	title string,
	body string,
	selectedDraftIDs []string,
) {
	ctx := r.Context()

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "suggestion_new_title", map[string]any{
		"SpaceName": output.Space.Name,
	})

	// フラッシュメッセージを取得
	flash := h.flashMgr.GetFlash(w, r)

	// ViewModelに変換
	spaceVM := viewmodel.NewSpace(output.Space)
	topicVM := viewmodel.NewTopic(output.Topic)
	draftPagesVM := viewmodel.NewDraftPagesForSuggestionNew(output.DraftPages)

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

	// サイドバーコンテンツを取得
	sidebarContent := h.sidebarHelper.Content(ctx, user.ID)

	layoutData := layouts.DefaultLayoutData{
		Meta:  meta,
		Flash: flash,
		Sidebar: components.SidebarData{
			CurrentPageName:   templates.PageNameSuggestionNew,
			SignedIn:          true,
			UserAtname:        user.Atname,
			SpaceIdentifier:   string(spaceIdentifier),
			JoinedTopics:      sidebarContent.JoinedTopics,
			DraftPages:        sidebarContent.DraftPages,
			HasMoreDraftPages: sidebarContent.HasMoreDraftPages,
		},
		BottomNav: components.BottomNavData{
			CurrentPageName: templates.PageNameSuggestionNew,
			SignedIn:        true,
			SpaceIdentifier: string(spaceIdentifier),
		},
	}

	err := layouts.Default(layoutData, content).Render(ctx, w)
	if err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

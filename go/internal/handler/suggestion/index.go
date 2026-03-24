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
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	suggestionpages "github.com/wikinoapp/wikino/go/internal/templates/pages/suggestion"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Index は編集提案一覧画面を表示します (GET /s/{space_identifier}/topics/{topic_number}/suggestions)
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// URLパラメータを取得
	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))
	topicNumberStr := chi.URLParam(r, "topic_number")

	topicNumber, err := strconv.ParseInt(topicNumberStr, 10, 32)
	if err != nil {
		handler.NotFound(w, r)
		return
	}

	// タブパラメータを取得（デフォルトはオープン）
	tab := r.URL.Query().Get("tab")
	showClosed := tab == "closed"

	// ログインユーザーを取得
	user := middleware.UserFromContext(ctx)
	var userID *model.UserID
	if user != nil {
		userID = &user.ID
	}

	// UseCaseでデータを取得
	output, err := h.getSuggestionListUsecase.Execute(ctx, usecase.GetSuggestionListInput{
		SpaceIdentifier: spaceIdentifier,
		TopicNumber:     int32(topicNumber),
		UserID:          userID,
		ShowClosed:      showClosed,
	})
	if err != nil {
		slog.ErrorContext(ctx, "編集提案一覧の取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if output == nil {
		handler.NotFound(w, r)
		return
	}

	// ViewModelに変換
	suggestions := viewmodel.NewSuggestionsForList(viewmodel.NewSuggestionForListInput{
		Suggestions: output.Suggestions,
		UserMap:     output.UserMap,
	})
	spaceVM := viewmodel.NewSpace(output.Space)
	topicVM := viewmodel.NewTopic(output.Topic)

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "suggestion_index_title", map[string]any{
		"TopicName": output.Topic.Name,
		"SpaceName": output.Space.Name,
	})

	// テンプレートをレンダリング
	content := suggestionpages.Index(suggestionpages.IndexData{
		Space:       spaceVM,
		Topic:       topicVM,
		Suggestions: suggestions,
		OpenCount:   output.OpenCount,
		ClosedCount: output.ClosedCount,
		ShowClosed:  showClosed,
	})

	signedIn := user != nil
	var userAtname string
	if user != nil {
		userAtname = user.Atname
	}

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,
		Sidebar: components.SidebarData{
			CurrentPageName: templates.PageNameSuggestionIndex,
			SignedIn:        signedIn,
			UserAtname:      userAtname,
			SpaceIdentifier: string(spaceIdentifier),
		},
		BottomNav: components.BottomNavData{
			CurrentPageName: templates.PageNameSuggestionIndex,
			SignedIn:        signedIn,
			SpaceIdentifier: string(spaceIdentifier),
		},
	}

	// ログイン済みの場合はサイドバーコンテンツを取得
	if user != nil {
		sidebarContent := h.sidebarHelper.Content(ctx, user.ID)
		layoutData.Sidebar.JoinedTopics = sidebarContent.JoinedTopics
		layoutData.Sidebar.DraftPages = sidebarContent.DraftPages
		layoutData.Sidebar.HasMoreDraftPages = sidebarContent.HasMoreDraftPages
	}

	err = layouts.Default(layoutData, content).Render(ctx, w)
	if err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

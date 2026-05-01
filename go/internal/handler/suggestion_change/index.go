package suggestion_change

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
	suggestionchangepages "github.com/wikinoapp/wikino/go/internal/templates/pages/suggestion_change"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Index は編集提案の変更差分を表示します (GET /s/{space_identifier}/suggestions/{suggestion_number}/changes)
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// URLパラメータを取得
	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))
	suggestionNumberStr := chi.URLParam(r, "suggestion_number")

	suggestionNumber, err := strconv.ParseInt(suggestionNumberStr, 10, 32)
	if err != nil {
		handler.NotFound(w, r)
		return
	}

	// ログインユーザーを取得
	user := middleware.UserFromContext(ctx)
	var userID *model.UserID
	if user != nil {
		userID = &user.ID
	}

	// UseCaseでデータを取得
	output, err := h.getSuggestionDetailUsecase.Execute(ctx, usecase.GetSuggestionDetailInput{
		SpaceIdentifier:  spaceIdentifier,
		SuggestionNumber: model.SuggestionNumber(suggestionNumber),
		UserID:           userID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "編集提案詳細の取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if output == nil {
		handler.NotFound(w, r)
		return
	}

	// ベースリビジョンを取得して差分を計算
	diffOutput, err := h.getSuggestionDiffUsecase.Execute(ctx, usecase.GetSuggestionDiffInput{
		SpaceID:         output.Space.ID,
		SuggestionPages: output.SuggestionPages,
	})
	if err != nil {
		slog.ErrorContext(ctx, "編集提案の差分取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// ViewModelに変換
	suggestionVM := viewmodel.NewSuggestionForDetail(viewmodel.NewSuggestionForDetailInput{
		Suggestion: output.Suggestion,
		UserMap:    output.UserMap,
	})
	suggestionPagesVM := viewmodel.NewSuggestionPagesForList(output.SuggestionPages)
	spaceVM := viewmodel.NewSpace(output.Space)
	topicVM := viewmodel.NewTopic(output.Topic)
	pageDiffsVM := viewmodel.NewSuggestionPageDiffs(viewmodel.NewSuggestionPageDiffsInput{
		SuggestionPages: output.SuggestionPages,
		BaseRevisions:   diffOutput.BaseRevisions,
		Pages:           output.Pages,
	})

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "suggestion_show_title", map[string]any{
		"SuggestionTitle":  output.Suggestion.Title,
		"SuggestionNumber": output.Suggestion.Number,
		"TopicName":        output.Topic.Name,
		"SpaceName":        output.Space.Name,
	})
	meta.CurrentSpaceIdentifier = string(spaceIdentifier)

	// 編集権限を判定（スペースメンバーかつオープンステータス）
	canEditSuggestionPages := output.SpaceMember != nil && output.Suggestion.Status == model.SuggestionStatusOpen

	// CSRFトークンを取得（編集・削除ボタンのフォームで必要な場合のみ）
	var csrfToken string
	if canEditSuggestionPages || output.CanAddSuggestionPage || output.CanRemoveSuggestionPage {
		csrfToken = middleware.GetCSRFTokenFromContext(ctx)
	}

	// テンプレートをレンダリング
	content := suggestionchangepages.Index(suggestionchangepages.IndexData{
		CSRFToken:               csrfToken,
		Space:                   spaceVM,
		Topic:                   topicVM,
		Suggestion:              suggestionVM,
		SuggestionPages:         suggestionPagesVM,
		PageDiffs:               pageDiffsVM,
		CanEditSuggestionPages:  canEditSuggestionPages,
		CanAddSuggestionPage:    output.CanAddSuggestionPage,
		CanRemoveSuggestionPage: output.CanRemoveSuggestionPage,
	})

	if err := suggestionhandler.RenderLayout(ctx, w, suggestionhandler.RenderLayoutInput{
		Cfg:             h.cfg,
		SidebarHelper:   h.sidebarHelper,
		User:            user,
		SpaceIdentifier: spaceIdentifier,
		CurrentPageName: templates.PageNameSuggestionChanges,
		Meta:            meta,
		Content:         content,
	}); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

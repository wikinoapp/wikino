package suggestion

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	suggestionpages "github.com/wikinoapp/wikino/go/internal/templates/pages/suggestion"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Show は編集提案詳細画面を表示します (GET /s/{space_identifier}/suggestions/{suggestion_number})
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
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

	// ViewModelに変換
	suggestionVM := viewmodel.NewSuggestionForDetail(viewmodel.NewSuggestionForDetailInput{
		Suggestion: output.Suggestion,
		UserMap:    output.UserMap,
	})
	commentsVM := viewmodel.NewSuggestionCommentsForList(viewmodel.NewSuggestionCommentsForListInput{
		Comments: output.Comments,
		UserMap:  output.UserMap,
	})
	suggestionPagesVM := viewmodel.NewSuggestionPagesForList(output.SuggestionPages)
	spaceVM := viewmodel.NewSpace(output.Space)
	topicVM := viewmodel.NewTopic(output.Topic)

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "suggestion_show_title", map[string]any{
		"SuggestionTitle":  output.Suggestion.Title,
		"SuggestionNumber": output.Suggestion.Number,
		"TopicName":        output.Topic.Name,
		"SpaceName":        output.Space.Name,
	})

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// 権限をチェック（オープンステータスかつ権限がある場合のみ）
	var canApply, canClose, canUpdateSuggestion, canUpdateSuggestionComment bool
	if output.SpaceMember != nil && output.Suggestion.Status == model.SuggestionStatusOpen {
		topicPolicy := policy.NewTopicPolicy(output.SpaceMember, output.TopicMember)
		canApply = topicPolicy.CanApplySuggestion(output.Suggestion)
		canClose = topicPolicy.CanCloseSuggestion(output.Suggestion)
		canUpdateSuggestion = topicPolicy.CanUpdateSuggestion(output.Suggestion)
		canUpdateSuggestionComment = topicPolicy.CanUpdateSuggestionComment(output.Suggestion)
	}

	// テンプレートをレンダリング
	content := suggestionpages.Show(suggestionpages.ShowData{
		CSRFToken:                  csrfToken,
		Space:                      spaceVM,
		Topic:                      topicVM,
		Suggestion:                 suggestionVM,
		Comments:                   commentsVM,
		SuggestionPages:            suggestionPagesVM,
		IsSpaceMember:              output.SpaceMember != nil,
		CanApply:                   canApply,
		CanClose:                   canClose,
		CanUpdateSuggestion:        canUpdateSuggestion,
		CanUpdateSuggestionComment: canUpdateSuggestionComment,
	})

	signedIn := user != nil
	var userAtname string
	if user != nil {
		userAtname = user.Atname
	}

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,
		Sidebar: components.SidebarData{
			CurrentPageName: templates.PageNameSuggestionShow,
			SignedIn:        signedIn,
			UserAtname:      userAtname,
			SpaceIdentifier: string(spaceIdentifier),
		},
		BottomNav: components.BottomNavData{
			CurrentPageName: templates.PageNameSuggestionShow,
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

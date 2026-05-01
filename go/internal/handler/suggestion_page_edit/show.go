package suggestion_page_edit

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
	suggestionpageeditpages "github.com/wikinoapp/wikino/go/internal/templates/pages/suggestion_page_edit"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Show は編集提案ページ編集の確認画面を表示します (GET /s/{space_identifier}/suggestions/{suggestion_number}/page_edits/{suggestion_page_id})
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
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

	// URLパラメータを取得（suggestion_page_id）
	suggestionPageIDStr := chi.URLParam(r, "suggestion_page_id")
	if suggestionPageIDStr == "" {
		handler.NotFound(w, r)
		return
	}
	suggestionPageID := model.SuggestionPageID(suggestionPageIDStr)

	// 編集提案詳細を取得
	detailOutput, err := h.getSuggestionDetailUsecase.Execute(ctx, usecase.GetSuggestionDetailInput{
		SpaceIdentifier:  spaceIdentifier,
		SuggestionNumber: suggestionNumber,
		UserID:           &user.ID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "編集提案詳細の取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if detailOutput == nil {
		handler.NotFound(w, r)
		return
	}

	// スペースメンバーでなければ403
	if detailOutput.SpaceMember == nil {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	spaceIdentVM := viewmodel.NewSpaceIdentifier(spaceIdentifier)

	// オープンステータスでなければ変更差分画面にリダイレクト
	if detailOutput.Suggestion.Status != model.SuggestionStatusOpen {
		changesPath := string(templates.SuggestionChangesPath(spaceIdentVM, int32(suggestionNumber)))
		http.Redirect(w, r, changesPath, http.StatusSeeOther)
		return
	}

	// 編集提案ページのタイトルを取得
	var suggestionPageTitle string
	found := false
	for _, sp := range detailOutput.SuggestionPages {
		if sp.ID == suggestionPageID {
			if sp.Title != nil {
				suggestionPageTitle = *sp.Title
			}
			found = true
			break
		}
	}
	if !found {
		handler.NotFound(w, r)
		return
	}

	// ViewModelに変換
	spaceVM := viewmodel.NewSpace(detailOutput.Space)
	topicVM := viewmodel.NewTopic(detailOutput.Topic)

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "suggestion_page_edit_confirm_title", map[string]any{
		"SpaceName": detailOutput.Space.Name,
	})
	meta.CurrentSpaceIdentifier = spaceIdentVM

	// テンプレートをレンダリング
	content := suggestionpageeditpages.Show(suggestionpageeditpages.ShowData{
		CSRFToken:           csrfToken,
		Space:               spaceVM,
		Topic:               topicVM,
		SuggestionNumber:    int32(suggestionNumber),
		SuggestionPageID:    string(suggestionPageID),
		SuggestionPageTitle: suggestionPageTitle,
	})

	// サイドバーコンテンツを取得
	sidebarContent := h.sidebarHelper.Content(ctx, user.ID)

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		Sidebar: components.SidebarData{
			CurrentPageName:   templates.PageNameSuggestionPageEditShow,
			SignedIn:          true,
			UserAtname:        user.Atname,
			SpaceIdentifier:   spaceIdentVM,
			JoinedTopics:      sidebarContent.JoinedTopics,
			DraftPages:        sidebarContent.DraftPages,
			HasMoreDraftPages: sidebarContent.HasMoreDraftPages,
		},
		BottomNav: components.BottomNavData{
			CurrentPageName: templates.PageNameSuggestionPageEditShow,
			SignedIn:        true,
			SpaceIdentifier: spaceIdentVM,
		},
	}

	err = layouts.Default(layoutData, content).Render(ctx, w)
	if err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

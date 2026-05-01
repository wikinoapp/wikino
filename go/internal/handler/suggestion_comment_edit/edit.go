package suggestion_comment_edit

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
	suggestioncommentpages "github.com/wikinoapp/wikino/go/internal/templates/pages/suggestion_comment"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Edit は編集提案コメント編集フォームを表示します (GET /s/{space_identifier}/suggestions/{suggestion_number}/comments/{comment_number}/edit)
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
	commentNumberStr := chi.URLParam(r, "comment_number")

	suggestionNumber, err := strconv.ParseInt(suggestionNumberStr, 10, 32)
	if err != nil {
		handler.NotFound(w, r)
		return
	}

	commentNumber, err := strconv.ParseInt(commentNumberStr, 10, 32)
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
	if !output.CanUpdateSuggestionComment {
		handler.NotFound(w, r)
		return
	}

	// コメントを取得
	commentOutput, err := h.getSuggestionCommentUsecase.Execute(ctx, usecase.GetSuggestionCommentInput{
		SuggestionID:  output.Suggestion.ID,
		CommentNumber: model.SuggestionCommentNumber(commentNumber),
		SpaceID:       output.Space.ID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "編集提案コメントの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if commentOutput.Comment == nil {
		handler.NotFound(w, r)
		return
	}

	h.renderEditForm(w, r, user, spaceIdentifier, output, commentOutput.Comment, nil, commentOutput.Comment.Body)
}

// renderEditForm は編集提案コメント編集フォームをレンダリングします
func (h *Handler) renderEditForm(
	w http.ResponseWriter,
	r *http.Request,
	user *model.User,
	spaceIdentifier model.SpaceIdentifier,
	output *usecase.GetSuggestionEditOutput,
	comment *model.SuggestionComment,
	formErrors *model.ValidationError,
	body string,
) {
	ctx := r.Context()

	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	spaceIdentVM := viewmodel.NewSpaceIdentifier(spaceIdentifier)

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "suggestion_comment_edit_title", map[string]any{
		"SuggestionNumber": output.Suggestion.Number,
		"TopicName":        output.Topic.Name,
		"SpaceName":        output.Space.Name,
	})
	meta.CurrentSpaceIdentifier = spaceIdentVM

	spaceVM := viewmodel.NewSpace(output.Space)
	topicVM := viewmodel.NewTopic(output.Topic)
	suggestionVM := viewmodel.NewSuggestionForDetail(viewmodel.NewSuggestionForDetailInput{
		Suggestion: output.Suggestion,
		UserMap:    output.UserMap,
	})
	commentVM := viewmodel.NewSuggestionCommentsForList(viewmodel.NewSuggestionCommentsForListInput{
		Comments: []*model.SuggestionComment{comment},
		UserMap:  output.UserMap,
	})[0]

	content := suggestioncommentpages.Edit(suggestioncommentpages.EditData{
		CSRFToken:  csrfToken,
		FormErrors: formErrors,
		Space:      spaceVM,
		Topic:      topicVM,
		Suggestion: suggestionVM,
		Comment:    commentVM,
		Body:       body,
	})

	sidebarContent := h.sidebarHelper.Content(ctx, user.ID)

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		Sidebar: components.SidebarData{
			CurrentPageName:   templates.PageNameSuggestionCommentEdit,
			SignedIn:          true,
			UserAtname:        user.Atname,
			SpaceIdentifier:   spaceIdentVM,
			JoinedTopics:      sidebarContent.JoinedTopics,
			DraftPages:        sidebarContent.DraftPages,
			HasMoreDraftPages: sidebarContent.HasMoreDraftPages,
		},
		BottomNav: components.BottomNavData{
			CurrentPageName: templates.PageNameSuggestionCommentEdit,
			SignedIn:        true,
			SpaceIdentifier: spaceIdentVM,
		},
	}

	err := layouts.Default(layoutData, content).Render(ctx, w)
	if err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

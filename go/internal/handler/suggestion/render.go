package suggestion

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/sidebar"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	suggestionpages "github.com/wikinoapp/wikino/go/internal/templates/pages/suggestion"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// RenderLayoutInput は編集提案関連ハンドラーで共有するレイアウトレンダリングの入力
type RenderLayoutInput struct {
	Cfg             *config.Config
	SidebarHelper   *sidebar.Helper
	User            *model.User
	SpaceIdentifier model.SpaceIdentifier
	CurrentPageName templates.PageName
	Meta            viewmodel.PageMeta
	Content         templ.Component
}

// RenderLayout は編集提案関連ページのレイアウト組み立てと最終レンダリングを行う。
// sidebar / BottomNav の組み立て・サイドバーコンテンツの取得・layouts.Default の呼び出しを一元化する。
func RenderLayout(ctx context.Context, w http.ResponseWriter, input RenderLayoutInput) error {
	signedIn := input.User != nil
	var userAtname string
	if input.User != nil {
		userAtname = input.User.Atname
	}

	layoutData := layouts.DefaultLayoutData{
		Meta: input.Meta,
		Sidebar: components.SidebarData{
			CurrentPageName: input.CurrentPageName,
			SignedIn:        signedIn,
			UserAtname:      userAtname,
			SpaceIdentifier: string(input.SpaceIdentifier),
		},
		BottomNav: components.BottomNavData{
			CurrentPageName: input.CurrentPageName,
			SignedIn:        signedIn,
			SpaceIdentifier: string(input.SpaceIdentifier),
		},
	}

	if input.User != nil {
		sidebarContent := input.SidebarHelper.Content(ctx, input.User.ID)
		layoutData.Sidebar.JoinedTopics = sidebarContent.JoinedTopics
		layoutData.Sidebar.DraftPages = sidebarContent.DraftPages
		layoutData.Sidebar.HasMoreDraftPages = sidebarContent.HasMoreDraftPages
	}

	if err := layouts.Default(layoutData, input.Content).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		return err
	}
	return nil
}

// RenderShowInput は RenderShow のための入力
type RenderShowInput struct {
	Cfg             *config.Config
	SidebarHelper   *sidebar.Helper
	User            *model.User
	SpaceIdentifier model.SpaceIdentifier
	Output          *usecase.GetSuggestionDetailOutput
	// ApplyError は反映処理のバリデーション結果を表示する場合にセットする。nil の場合はエラー表示を行わない
	ApplyError *viewmodel.SuggestionApplyError
}

// RenderShow は編集提案詳細ページをレンダリングする。
// 通常の Show / 反映失敗時の再描画の両方で使用される。
func RenderShow(ctx context.Context, w http.ResponseWriter, input RenderShowInput) error {
	output := input.Output

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

	meta := viewmodel.DefaultPageMeta(ctx, input.Cfg)
	meta.SetTitleWithoutSuffix(ctx, "suggestion_show_title", map[string]any{
		"SuggestionTitle":  output.Suggestion.Title,
		"SuggestionNumber": output.Suggestion.Number,
		"TopicName":        output.Topic.Name,
		"SpaceName":        output.Space.Name,
	})
	meta.CurrentSpaceIdentifier = string(input.SpaceIdentifier)

	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	var canApply, canClose, canUpdateSuggestion, canUpdateSuggestionComment bool
	if output.Suggestion.Status == model.SuggestionStatusOpen {
		canApply = output.CanApplySuggestion
		canClose = output.CanCloseSuggestion
		canUpdateSuggestion = output.CanUpdateSuggestion
		canUpdateSuggestionComment = output.CanUpdateSuggestionComment
	}

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
		ApplyError:                 input.ApplyError,
	})

	return RenderLayout(ctx, w, RenderLayoutInput{
		Cfg:             input.Cfg,
		SidebarHelper:   input.SidebarHelper,
		User:            input.User,
		SpaceIdentifier: input.SpaceIdentifier,
		CurrentPageName: templates.PageNameSuggestionShow,
		Meta:            meta,
		Content:         content,
	})
}

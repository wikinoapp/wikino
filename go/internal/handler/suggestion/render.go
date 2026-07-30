package suggestion

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	suggestionpages "github.com/wikinoapp/wikino/go/internal/templates/pages/suggestion"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// RenderLayoutInput is the input for the layout rendering shared by suggestion-related handlers.
// BreadcrumbHeader carries the breadcrumb header, which the layout renders outside <main>.
//
// [Ja] RenderLayoutInput は編集提案関連ハンドラーで共有するレイアウトレンダリングの入力。
// BreadcrumbHeader はパンくずヘッダーを保持し、レイアウトが <main> の外で描画する。
type RenderLayoutInput struct {
	User             *model.User
	SpaceIdentifier  model.SpaceIdentifier
	CurrentPageName  templates.PageName
	Meta             viewmodel.PageMeta
	BreadcrumbHeader components.BreadcrumbHeaderData
	Content          templ.Component
}

// RenderLayout assembles the shared layout for suggestion-related pages and renders it. It
// centralizes building the global-nav state, the breadcrumb header and the layouts.Default call.
//
// [Ja] RenderLayout は編集提案関連ページのレイアウト組み立てと最終レンダリングを行う。
// グローバルナビ状態・パンくずヘッダーの組み立てと layouts.Default の呼び出しを一元化する。
func RenderLayout(ctx context.Context, w http.ResponseWriter, input RenderLayoutInput) error {
	signedIn := input.User != nil
	var userAtname string
	if input.User != nil {
		userAtname = input.User.Atname
	}

	layoutData := layouts.DefaultLayoutData{
		Meta: input.Meta,
		GlobalNav: components.GlobalNavData{
			CurrentPageName: input.CurrentPageName,
			SignedIn:        signedIn,
			UserAtname:      userAtname,
			SpaceIdentifier: viewmodel.NewSpaceIdentifier(input.SpaceIdentifier),
		},
		BreadcrumbHeader: input.BreadcrumbHeader,
	}

	if err := layouts.Default(layoutData, input.Content).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		return err
	}
	return nil
}

// suggestionBreadcrumbHeaderMaxWidthClass is the content width shared by every suggestion-family
// screen. The breadcrumb header takes the same width so it lines up with the body container.
//
// [Ja] suggestionBreadcrumbHeaderMaxWidthClass は編集提案系の全画面で共通の本文幅。パンくずヘッダーを本文
// コンテナと揃えるため、ヘッダーにも同じ幅を渡す。
const suggestionBreadcrumbHeaderMaxWidthClass = "max-w-3xl"

// spaceBreadcrumbHeaderData builds the breadcrumb header up to the space (home › space).
//
// [Ja] spaceBreadcrumbHeaderData はスペースまでのパンくずヘッダー (ホーム › スペース) を組み立てる。
func spaceBreadcrumbHeaderData(ctx context.Context, space viewmodel.Space) components.BreadcrumbHeaderData {
	return components.BreadcrumbHeaderData{
		MaxWidthClass: suggestionBreadcrumbHeaderMaxWidthClass,
		Items: []components.BreadcrumbItem{
			{
				Path:      templates.HomePath(),
				IconName:  "house-regular",
				AriaLabel: i18n.T(ctx, "breadcrumb_home"),
			},
			{
				Label: space.Name,
				Path:  templates.SpacePath(space.Identifier),
			},
		},
	}
}

// topicBreadcrumbHeaderData builds the breadcrumb header up to the topic (home › space › topic).
//
// [Ja] topicBreadcrumbHeaderData はトピックまでのパンくずヘッダー (ホーム › スペース › トピック) を
// 組み立てる。
func topicBreadcrumbHeaderData(ctx context.Context, space viewmodel.Space, topic viewmodel.Topic) components.BreadcrumbHeaderData {
	data := spaceBreadcrumbHeaderData(ctx, space)
	data.Items = append(data.Items, components.BreadcrumbItem{
		Label:    topic.Name,
		Path:     templates.TopicPath(space.Identifier, topic.Number),
		IconName: topic.IconName,
	})
	return data
}

// DetailBreadcrumbHeaderData builds the breadcrumb header (home › space › topic › suggestion #N)
// for the screens nested under a suggestion: its edit form, the change diff, the comment edit
// form, the page add form and the page edit confirmation. The layout renders the header, so those
// handlers supply it from here rather than repeating the same four crumbs.
//
// [Ja] DetailBreadcrumbHeaderData は編集提案の配下にある画面 (編集提案の編集 / 変更差分 / コメント編集 /
// ページ追加 / ページ編集の確認) のパンくずヘッダー (ホーム › スペース › トピック › 編集提案 #N) を
// 組み立てる。描画するのはレイアウトのため、各ハンドラーで同じ 4 項目を繰り返さずここから供給する。
func DetailBreadcrumbHeaderData(
	ctx context.Context,
	space viewmodel.Space,
	topic viewmodel.Topic,
	suggestionNumber int32,
) components.BreadcrumbHeaderData {
	data := topicBreadcrumbHeaderData(ctx, space, topic)
	data.Items = append(data.Items, components.BreadcrumbItem{
		Label: fmt.Sprintf("%s #%d", i18n.T(ctx, "suggestion_show_breadcrumb"), suggestionNumber),
		Path:  templates.SuggestionShowPath(space.Identifier, suggestionNumber),
	})
	return data
}

// RenderShowInput は RenderShow のための入力
type RenderShowInput struct {
	Cfg             *config.Config
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
	meta.CurrentSpaceIdentifier = viewmodel.NewSpaceIdentifier(input.SpaceIdentifier)

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

	// The suggestion detail is the current page, so the trailing crumb links back to the suggestion
	// list of the topic instead of to the suggestion itself.
	//
	// [Ja] 編集提案詳細は現在地のため、末尾のパンくずは編集提案自身ではなくトピックの編集提案一覧へ
	// リンクする。
	breadcrumbHeader := topicBreadcrumbHeaderData(ctx, spaceVM, topicVM)
	breadcrumbHeader.Items = append(breadcrumbHeader.Items, components.BreadcrumbItem{
		Label: i18n.T(ctx, "suggestion_show_breadcrumb"),
		Path:  templates.SuggestionListPath(spaceVM.Identifier, topicVM.Number),
	})

	return RenderLayout(ctx, w, RenderLayoutInput{
		User:             input.User,
		SpaceIdentifier:  input.SpaceIdentifier,
		CurrentPageName:  templates.PageNameSuggestionShow,
		Meta:             meta,
		BreadcrumbHeader: breadcrumbHeader,
		Content:          content,
	})
}

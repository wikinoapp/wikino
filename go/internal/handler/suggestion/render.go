package suggestion

import (
	"context"
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

// RenderLayoutInputは編集提案関連ハンドラーで共有するレイアウトレンダリングの入力。
// BreadcrumbHeaderはパンくずヘッダーを保持し、レイアウトが <main> の外で描画する。
type RenderLayoutInput struct {
	User             *model.User
	SpaceIdentifier  model.SpaceIdentifier
	CurrentPageName  templates.PageName
	Meta             viewmodel.PageMeta
	BreadcrumbHeader components.BreadcrumbHeaderData
	Content          templ.Component
}

// RenderLayoutは編集提案関連ページのレイアウト組み立てと最終レンダリングを行う。
// グローバルナビ状態・パンくずヘッダーの組み立てとlayouts.Defaultの呼び出しを一元化する。
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

// suggestionBreadcrumbHeaderMaxWidthClassは編集提案系の全画面で共通の本文幅。パンくずヘッダーを本文
// コンテナと揃えるため、ヘッダーにも同じ幅を渡す。
const suggestionBreadcrumbHeaderMaxWidthClass = "max-w-3xl"

// spaceBreadcrumbHeaderDataはスペースまでのパンくずヘッダー (ホーム › スペース) を組み立てる。
// 編集提案の一覧と詳細は未ログインでも到達できるため、signedInには閲覧者の実際の状態を渡し、認証必須の
// 画面はtrueを渡す。
func spaceBreadcrumbHeaderData(ctx context.Context, space viewmodel.Space, signedIn bool) components.BreadcrumbHeaderData {
	return components.BreadcrumbHeaderData{
		MaxWidthClass: suggestionBreadcrumbHeaderMaxWidthClass,
		Items: append(components.HomeBreadcrumbItems(ctx, signedIn), components.BreadcrumbItem{
			Label: space.Name,
			Path:  templates.SpacePath(space.Identifier),
		}),
	}
}

// topicBreadcrumbHeaderDataはトピックまでのパンくずヘッダー (ホーム › スペース › トピック) を
// 組み立てる。
func topicBreadcrumbHeaderData(ctx context.Context, space viewmodel.Space, topic viewmodel.Topic, signedIn bool) components.BreadcrumbHeaderData {
	data := spaceBreadcrumbHeaderData(ctx, space, signedIn)
	data.Items = append(data.Items, components.BreadcrumbItem{
		Label:    topic.Name,
		Path:     templates.TopicPath(space.Identifier, topic.Number),
		IconName: topic.IconName,
	})
	return data
}

// suggestionListBreadcrumbHeaderDataはトピックの編集提案一覧までのパンくずヘッダー
// (ホーム › スペース › トピック › 編集提案一覧) を組み立てる。編集提案詳細と、編集提案の配下にある
// 各画面はここから続けるため、編集提案は常に所属する一覧の下に置かれ、2つの経路が編集提案の位置に
// ついて食い違うことがない。
func suggestionListBreadcrumbHeaderData(ctx context.Context, space viewmodel.Space, topic viewmodel.Topic, signedIn bool) components.BreadcrumbHeaderData {
	data := topicBreadcrumbHeaderData(ctx, space, topic, signedIn)
	data.Items = append(data.Items, components.BreadcrumbItem{
		Label: i18n.T(ctx, "suggestion_index_breadcrumb"),
		Path:  templates.SuggestionListPath(space.Identifier, topic.Number),
	})
	return data
}

// DetailBreadcrumbHeaderDataは編集提案の配下にある画面 (編集提案の編集 / 変更差分 / コメント編集 /
// ページ追加 / ページ編集の確認) のパンくずヘッダー
// (ホーム › スペース › トピック › 編集提案一覧 › 編集提案のタイトル) を組み立てる。描画するのは
// レイアウトのため、各ハンドラーで同じ項目を繰り返さずここから供給する。変更差分は未ログインでも
// 到達できるため、そこではsignedInに閲覧者の実際の状態を渡し、認証必須の画面はtrueを渡す。
//
// suggestionTitleは詳細画面が自身を表すのと同じ名前を使う。これにより1つのURLが画面をまたいで
// 1つの名前を持ち、同じ項目列から生成するBreadcrumbList JSON-LDでも一致する。
func DetailBreadcrumbHeaderData(
	ctx context.Context,
	space viewmodel.Space,
	topic viewmodel.Topic,
	suggestionNumber int32,
	suggestionTitle string,
	signedIn bool,
) components.BreadcrumbHeaderData {
	data := suggestionListBreadcrumbHeaderData(ctx, space, topic, signedIn)
	data.Items = append(data.Items, components.BreadcrumbItem{
		Label: suggestionTitle,
		Path:  templates.SuggestionShowPath(space.Identifier, suggestionNumber),
	})
	return data
}

// RenderShowInputはRenderShowのための入力。
// スペース識別子やページ内リンクの組み立てにはOutputに含まれる保存済みの値を使うため、
// URLパラメータ由来の識別子は受け取らない。
type RenderShowInput struct {
	Cfg    *config.Config
	User   *model.User
	Output *usecase.GetSuggestionDetailOutput
	// ApplyErrorは反映処理のバリデーション結果を表示する場合にセットする。nilの場合はエラー表示を行わない
	ApplyError *viewmodel.SuggestionApplyError
}

// RenderShowは編集提案詳細ページをレンダリングする。
// 通常のShow / 反映失敗時の再描画の両方で使用される。
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

	// URLではなく保存済みの識別子からリンクを組み立て、正規URLを1画面1アドレスに
	// 集約する。
	spaceIdentVM := spaceVM.Identifier

	meta := viewmodel.DefaultPageMeta(ctx, input.Cfg)
	meta.SetTitleWithoutSuffix(ctx, "suggestion_show_title", map[string]any{
		"SuggestionTitle":  output.Suggestion.Title,
		"SuggestionNumber": output.Suggestion.Number,
		"TopicName":        output.Topic.Name,
		"SpaceName":        output.Space.Name,
	})
	meta.OGURL = input.Cfg.AppURL() + string(templates.SuggestionShowPath(spaceIdentVM, suggestionVM.Number))
	meta.CurrentSpaceIdentifier = spaceIdentVM

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

	// 編集提案詳細は現在地のため、経路は編集提案一覧を通り、提案タイトルの非リンクな現在項目で
	// 締める。本画面の配下にある画面はDetailBreadcrumbHeaderDataで同じ一覧の項目から続き、そこでは
	// タイトルがリンクになる。
	breadcrumbHeader := suggestionListBreadcrumbHeaderData(ctx, spaceVM, topicVM, input.User != nil)
	breadcrumbHeader.Items = append(breadcrumbHeader.Items, components.BreadcrumbItem{
		Label:     suggestionVM.Title,
		IsCurrent: true,
	})

	// 編集提案詳細は公開画面で自己参照canonicalを宣言するため、同じ項目列から作る
	// BreadcrumbList JSON-LDを有効にする。未ログインの閲覧者は公開スペースから始め、ログイン済みの
	// 閲覧者には /homeも含める。
	breadcrumbHeader.StructuredDataBaseURL = input.Cfg.AppURL()

	return RenderLayout(ctx, w, RenderLayoutInput{
		User:             input.User,
		SpaceIdentifier:  output.Space.Identifier,
		CurrentPageName:  templates.PageNameSuggestionShow,
		Meta:             meta,
		BreadcrumbHeader: breadcrumbHeader,
		Content:          content,
	})
}

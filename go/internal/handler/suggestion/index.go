package suggestion

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	suggestionpages "github.com/wikinoapp/wikino/go/internal/templates/pages/suggestion"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Indexは編集提案一覧画面を表示します (GET /s/{space_identifier}/topics/{topic_number}/suggestions)
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

	// タブパラメータを取得 (デフォルトはオープン)
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

	// URLではなく保存済みの識別子からリンクを組み立て、正規URLを1画面1アドレスに
	// 集約する。
	spaceIdentVM := spaceVM.Identifier

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	titleKey := "suggestion_index_title"
	if showClosed {
		titleKey = "suggestion_index_closed_title"
	}
	meta.SetTitleWithoutSuffix(ctx, titleKey, map[string]any{
		"TopicName": output.Topic.Name,
		"SpaceName": output.Space.Name,
	})
	meta.OGURL = h.cfg.AppURL() + string(templates.SuggestionListTabPath(spaceIdentVM, topicVM.Number, showClosed))
	meta.CurrentSpaceIdentifier = spaceIdentVM

	// テンプレートをレンダリング
	content := suggestionpages.Index(suggestionpages.IndexData{
		Space:               spaceVM,
		Topic:               topicVM,
		Suggestions:         suggestions,
		OpenCount:           output.OpenCount,
		ClosedCount:         output.ClosedCount,
		ShowClosed:          showClosed,
		CanCreateSuggestion: output.CanCreateSuggestion,
	})

	// 編集提案一覧は現在地のため、経路はトピックを通り、一覧自身の非リンクな現在項目で締める。
	// 編集提案配下の画面もsuggestionListBreadcrumbHeaderDataで同じ一覧の項目を通るため、1つの
	// URLが階層上で1つの位置を持つ。
	//
	// 一覧は公開画面で自己参照canonicalを宣言するため、同じ項目列から作るBreadcrumbList JSON-LDを
	// 有効にする。ステータスタブは載っている編集提案が異なるだけで階層上の位置は同じなので、経路を
	// 共有する。
	breadcrumbHeader := topicBreadcrumbHeaderData(ctx, spaceVM, topicVM, user != nil)
	breadcrumbHeader.Items = append(breadcrumbHeader.Items, components.BreadcrumbItem{
		Label:     i18n.T(ctx, "suggestion_index_breadcrumb"),
		IsCurrent: true,
	})
	breadcrumbHeader.StructuredDataBaseURL = h.cfg.AppURL()

	if err := RenderLayout(ctx, w, RenderLayoutInput{
		User:             user,
		SpaceIdentifier:  output.Space.Identifier,
		CurrentPageName:  templates.PageNameSuggestionIndex,
		Meta:             meta,
		BreadcrumbHeader: breadcrumbHeader,
		Content:          content,
	}); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

package page

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
	pagepages "github.com/wikinoapp/wikino/go/internal/templates/pages/page"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// zenModeCookieName is the cookie holding the editor's Zen mode state ("1" = on; off is the
// cookie's absence). It is written by web/zen-mode.ts, so it is not HttpOnly; keep the name in
// sync with that script.
//
// [Ja] zenModeCookieName はエディタの Zenモード状態を保持するクッキー ("1" で ON。OFF は
// クッキーなし)。web/zen-mode.ts が書き込むため HttpOnly ではない。名前は同スクリプトと
// 同期させること。
const zenModeCookieName = "wikino_zen_mode"

// zenModeFromRequest reads the Zen mode state from the request cookie.
// [Ja] zenModeFromRequest はリクエストのクッキーから Zenモード状態を読み取ります。
func zenModeFromRequest(r *http.Request) bool {
	cookie, err := r.Cookie(zenModeCookieName)
	return err == nil && cookie.Value == "1"
}

// Edit はページ編集フォームを表示します (GET /s/{space_identifier}/pages/{page_number}/edit)
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
	pageNumberStr := chi.URLParam(r, "page_number")

	pageNumber, err := strconv.ParseInt(pageNumberStr, 10, 32)
	if err != nil {
		handler.NotFound(w, r)
		return
	}

	// UseCaseでデータを取得
	output, err := h.getPageDetailUC.Execute(ctx, usecase.GetPageDetailInput{
		SpaceIdentifier:       spaceIdentifier,
		PageNumber:            int32(pageNumber),
		UserID:                user.ID,
		IncludeDraftPages:     true,
		IncludeDraftRevisions: true,
	})
	if err != nil {
		slog.ErrorContext(ctx, "ページ詳細の取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if output == nil {
		handler.NotFound(w, r)
		return
	}

	// 認可チェック
	if !output.CanUpdatePage {
		handler.NotFound(w, r)
		return
	}

	// 編集画面用のリンクデータを取得
	linkData, err := h.getEditLinkDataUC.Execute(ctx, usecase.GetEditLinkDataInput{
		Page:              output.Page,
		DraftPage:         output.DraftPage,
		SpaceID:           output.Space.ID,
		CurrentPage:       1,
		LinkLimit:         viewmodel.LinkLimit,
		BacklinkLimit:     viewmodel.BacklinkLimit,
		PageBacklinkLimit: viewmodel.PageBacklinkLimit,
	})
	if err != nil {
		slog.ErrorContext(ctx, "リンクデータの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	linkResult := buildEditLinkResult(linkData, spaceIdentifier, 1, output.Page)

	// 編集画面用のページViewModelを生成
	pageVM := viewmodel.NewPageForEdit(output.Page, output.DraftPage)
	topicVM := viewmodel.NewTopic(output.Topic)

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	spaceIdentVM := viewmodel.NewSpaceIdentifier(spaceIdentifier)

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "page_edit_title", map[string]any{
		"SpaceName": output.Space.Name,
	})
	meta.CurrentSpaceIdentifier = spaceIdentVM

	// テンプレートをレンダリング
	spaceVM := viewmodel.NewSpace(output.Space)

	manualSaveURL := string(templates.PageDraftPageRevisionPath(spaceIdentVM, int32(output.Page.Number)))

	editData := pagepages.EditPageData{
		CSRFToken:               csrfToken,
		Page:                    pageVM,
		Space:                   spaceVM,
		Topic:                   topicVM,
		LinkList:                linkResult.LinkList,
		BacklinkList:            linkResult.BacklinkList,
		ManualSaveURL:           manualSaveURL,
		CreateSuggestionSaveURL: manualSaveURL + "?redirect_to=suggestion_new",
		// The editor stays within a single space, so omit the space label on each draft card.
		// [Ja] 編集画面は同一スペース内のため、各下書きカードのスペースラベルを省く。
		DraftPages:     viewmodel.NewCardLinkDraftPagesWithoutSpace(output.DraftPages),
		DraftRevisions: viewmodel.NewDraftPageRevisions(output.DraftPageRevisions, output.DraftPageRevisionTotalCount),
		ZenMode:        zenModeFromRequest(r),
	}

	if output.Suggestion != nil && output.DraftPage != nil && output.DraftPage.SuggestionPageID != nil {
		editData.SuggestionNumber = int32(output.Suggestion.Number)
		editData.SuggestionURL = string(templates.SuggestionPagePath(spaceIdentVM, int32(output.Suggestion.Number), string(*output.DraftPage.SuggestionPageID)))
		editData.SuggestionShowURL = string(templates.SuggestionShowPath(spaceIdentVM, int32(output.Suggestion.Number)))
	}

	content := pagepages.Edit(editData)

	// The page editor hides the global sidebar: the left draft list column takes over in-screen
	// draft navigation, so HideSidebar is set and no Sidebar data is built.
	// [Ja] ページ編集画面はグローバルサイドバーを非表示にする。左カラムの下書き一覧が画面内の下書き
	// ナビゲーションを担うため、HideSidebar を立て、Sidebar データは構築しない。
	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		HideFooter:  true,
		HideSidebar: true,
		BottomNav: components.BottomNavData{
			CurrentPageName:   templates.PageNamePageEdit,
			SignedIn:          true,
			SpaceIdentifier:   spaceIdentVM,
			HideSidebarToggle: true,
		},
	}

	err = layouts.Default(layoutData, content).Render(ctx, w)
	if err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// editLinkResult はリンク一覧・バックリンク一覧のViewModel
type editLinkResult struct {
	LinkList     viewmodel.LinkList
	BacklinkList viewmodel.BacklinkList
}

// buildEditLinkResult はUseCaseのリンクデータ出力をViewModelに変換します
func buildEditLinkResult(linkData *usecase.GetEditLinkDataOutput, spaceIdentifier model.SpaceIdentifier, currentPage int32, pg *model.Page) *editLinkResult {
	backlinksPerPage := make(map[model.PageID]*viewmodel.PageSliceWithCount, len(linkData.BacklinksPerPage))
	for pageID, backlinks := range linkData.BacklinksPerPage {
		backlinksPerPage[pageID] = &viewmodel.PageSliceWithCount{
			Pages:      backlinks.Pages,
			TotalCount: backlinks.TotalCount,
		}
	}

	editLinkData := viewmodel.BuildEditLinkData(viewmodel.BuildEditLinkDataInput{
		LinkedPages:       linkData.LinkedPages,
		LinkedTotalCount:  linkData.LinkedTotalCount,
		BacklinksPerPage:  backlinksPerPage,
		PageBacklinks:     linkData.PageBacklinks,
		PageBacklinkCount: linkData.PageBacklinkCount,
		Topics:            linkData.LinkTopics,
		SpaceIdentifier:   spaceIdentifier,
		PageNumber:        int32(pg.Number),
		CurrentPage:       currentPage,
	})

	return &editLinkResult{
		LinkList:     editLinkData.LinkList,
		BacklinkList: editLinkData.BacklinkList,
	}
}

package page

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	pagepages "github.com/wikinoapp/wikino/go/internal/templates/pages/page"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

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
		http.NotFound(w, r)
		return
	}

	// UseCaseでデータを取得
	output, err := h.getPageDetailUC.Execute(ctx, usecase.GetPageDetailInput{
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pageNumber),
		UserID:          user.ID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "ページ詳細の取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if output == nil {
		http.NotFound(w, r)
		return
	}

	// 認可チェック
	topicPolicy := policy.NewTopicPolicy(output.SpaceMember, output.TopicMember)
	if !topicPolicy.CanUpdatePage(output.Page) {
		http.NotFound(w, r)
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

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "page_edit_title")

	// フラッシュメッセージを取得
	flash := h.flashMgr.GetFlash(w, r)

	// テンプレートをレンダリング
	spaceVM := viewmodel.NewSpace(output.Space)

	content := pagepages.Edit(pagepages.EditPageData{
		CSRFToken:     csrfToken,
		Page:          pageVM,
		Space:         spaceVM,
		Topic:         topicVM,
		LinkList:      linkResult.LinkList,
		BacklinkList:  linkResult.BacklinkList,
		ManualSaveURL: string(templates.PageDraftPageRevisionPath(spaceIdentifier.String(), int32(output.Page.Number))),
	})

	// サイドバーコンテンツを取得
	sidebarContent := h.sidebarHelper.Content(ctx, user.ID)

	layoutData := layouts.DefaultLayoutData{
		Meta:       meta,
		Flash:      flash,
		HideFooter: true,
		Sidebar: components.SidebarData{
			CurrentPageName:   templates.PageNamePageEdit,
			SignedIn:          true,
			UserAtname:        user.Atname,
			SpaceIdentifier:   string(spaceIdentifier),
			JoinedTopics:      sidebarContent.JoinedTopics,
			DraftPages:        sidebarContent.DraftPages,
			HasMoreDraftPages: sidebarContent.HasMoreDraftPages,
		},
		BottomNav: components.BottomNavData{
			CurrentPageName: templates.PageNamePageEdit,
			SignedIn:        true,
			SpaceIdentifier: string(spaceIdentifier),
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

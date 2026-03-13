package page

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/sidebar"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	pagepages "github.com/wikinoapp/wikino/go/internal/templates/pages/page"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Update はページを公開します (PATCH /s/{space_identifier}/pages/{page_number})
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
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

	// フォームデータを取得
	title := r.FormValue("title")
	body := r.FormValue("body")

	// バリデーション
	validationResult := h.updateValidator.Validate(ctx, validator.PageUpdateValidatorInput{
		Title:           title,
		PageID:          output.Page.ID,
		TopicID:         output.Page.TopicID,
		SpaceID:         output.Space.ID,
		SpaceIdentifier: spaceIdentifier,
	})

	if validationResult.FormErrors.HasErrors() {
		h.renderEditWithErrors(w, r, spaceIdentifier, output, title, body, validationResult.FormErrors)
		return
	}

	// Titleをポインタに変換（空文字列の場合はnil）
	var titlePtr *string
	if title != "" {
		titlePtr = &title
	}

	// ページ公開ユースケースを実行
	input := usecase.PublishPageInput{
		SpaceID:          output.Space.ID,
		PageID:           output.Page.ID,
		SpaceMemberID:    output.SpaceMember.ID,
		TopicID:          output.Page.TopicID,
		Title:            titlePtr,
		Body:             body,
		SpaceIdentifier:  spaceIdentifier,
		CurrentTopicName: output.Topic.Name,
	}

	// DraftPageが存在する場合はDraftPageIDを設定
	if output.DraftPage != nil {
		input.DraftPageID = output.DraftPage.ID
	}

	_, err = h.publishPageUC.Execute(ctx, input)
	if err != nil {
		slog.ErrorContext(ctx, "ページの公開に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// フラッシュメッセージを設定してリダイレクト
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_page_saved"))
	pagePath := fmt.Sprintf("/s/%s/pages/%d", string(spaceIdentifier), output.Page.Number)
	http.Redirect(w, r, pagePath, http.StatusSeeOther)
}

// renderEditWithErrors はバリデーションエラー時に編集画面を再表示します
func (h *Handler) renderEditWithErrors(
	w http.ResponseWriter,
	r *http.Request,
	spaceIdentifier model.SpaceIdentifier,
	output *usecase.GetPageDetailOutput,
	title string,
	body string,
	formErrors *session.FormErrors,
) {
	ctx := r.Context()

	// ViewModelを生成
	pageVM := viewmodel.NewPageFromFormInput(title, body, output.Page.Number)
	spaceVM := viewmodel.NewSpace(output.Space)
	topicVM := viewmodel.NewTopic(output.Topic)

	// リンクデータを取得
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

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "page_edit_title")

	content := pagepages.Edit(pagepages.EditPageData{
		CSRFToken:     csrfToken,
		FormErrors:    formErrors,
		Page:          pageVM,
		Space:         spaceVM,
		Topic:         topicVM,
		LinkList:      linkResult.LinkList,
		BacklinkList:  linkResult.BacklinkList,
		ManualSaveURL: string(templates.PageDraftPagePath(spaceIdentifier.String(), int32(output.Page.Number))),
	})

	currentUser := middleware.UserFromContext(ctx)
	var userAtname string
	var sidebarContent sidebar.SidebarContent
	if currentUser != nil {
		userAtname = currentUser.Atname
		sidebarContent = h.sidebarHelper.Content(ctx, currentUser.ID)
	}

	layoutData := layouts.DefaultLayoutData{
		Meta:       meta,
		HideFooter: true,
		Sidebar: components.SidebarData{
			CurrentPageName:   templates.PageNamePageEdit,
			SignedIn:          currentUser != nil,
			UserAtname:        userAtname,
			SpaceIdentifier:   string(spaceIdentifier),
			JoinedTopics:      sidebarContent.JoinedTopics,
			DraftPages:        sidebarContent.DraftPages,
			HasMoreDraftPages: sidebarContent.HasMoreDraftPages,
		},
		BottomNav: components.BottomNavData{
			CurrentPageName: templates.PageNamePageEdit,
			SignedIn:        currentUser != nil,
			SpaceIdentifier: string(spaceIdentifier),
		},
	}

	w.WriteHeader(http.StatusUnprocessableEntity)

	err = layouts.Default(layoutData, content).Render(ctx, w)
	if err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
	}
}

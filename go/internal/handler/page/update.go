package page

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
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	pagepages "github.com/wikinoapp/wikino/go/internal/templates/pages/page"
	"github.com/wikinoapp/wikino/go/internal/usecase"
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
		handler.NotFound(w, r)
		return
	}

	// フォームデータを取得
	title := r.FormValue("title")
	body := r.FormValue("body")

	// UseCase を実行
	publishOutput, err := h.publishPageUC.Execute(ctx, usecase.PublishPageInput{
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pageNumber),
		UserID:          user.ID,
		Title:           title,
		Body:            body,
	})
	if err != nil {
		h.handleUpdateError(w, r, err, user, spaceIdentifier, int32(pageNumber), title, body)
		return
	}

	// フラッシュメッセージを設定してリダイレクト
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_page_saved"))
	pagePath := string(templates.PagePath(viewmodel.NewSpaceIdentifier(spaceIdentifier), viewmodel.PageNumber(publishOutput.Page.Number)))
	http.Redirect(w, r, pagePath, http.StatusSeeOther)
}

func (h *Handler) handleUpdateError(w http.ResponseWriter, r *http.Request, err error, user *model.User, spaceIdentifier model.SpaceIdentifier, pageNumber int32, title, body string) {
	ctx := r.Context()

	if ve := model.AsValidationError(err); ve != nil {
		// バリデーションエラー → フォーム再描画
		output, getErr := h.getPageDetailUC.Execute(ctx, usecase.GetPageDetailInput{
			SpaceIdentifier:   spaceIdentifier,
			PageNumber:        pageNumber,
			UserID:            user.ID,
			IncludeDraftPages: true,
		})
		if getErr != nil || output == nil {
			slog.ErrorContext(ctx, "フォーム再表示用データの取得に失敗", "error", getErr, "original_error", ve.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		h.renderEditWithErrors(w, r, output, title, body, ve)
		return
	}

	if ae := model.AsAppError(err); ae != nil {
		switch ae.Code {
		case model.AppErrCodeResourceNotFound, model.AppErrCodeForbidden:
			handler.NotFound(w, r)
		default:
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	slog.ErrorContext(ctx, "ページの公開に失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// renderEditWithErrors re-renders the editor on a validation error. Space-aware links are built
// from the persisted identifier in output, so it deliberately does not take one derived from URL
// parameters.
//
// [Ja] renderEditWithErrors はバリデーションエラー時に編集画面を再表示します。
// スペース識別子を含むリンクの組み立てには output に含まれる保存済みの値を使うため、
// URL パラメータ由来の識別子は受け取りません。
func (h *Handler) renderEditWithErrors(
	w http.ResponseWriter,
	r *http.Request,
	output *usecase.GetPageDetailOutput,
	title string,
	body string,
	formErrors *model.ValidationError,
) {
	ctx := r.Context()

	// ViewModelを生成
	pageVM := viewmodel.NewPageFromFormInput(title, body, output.Page.Number)
	spaceVM := viewmodel.NewSpace(output.Space)
	topicVM := viewmodel.NewTopic(output.Topic)

	// Build links from the stored identifier, not from the URL, so that every link on the screen uses
	// the same form.
	//
	// [Ja] URL ではなく保存済みの識別子からリンクを組み立て、画面内のリンクの表記を揃える。
	spaceIdentVM := spaceVM.Identifier

	// Re-rendering the editor after a validation error starts every listing at its first page: the
	// submitted form carries no related-page pagination state.
	//
	// [Ja] バリデーションエラー後の編集画面の再描画では、各一覧を 1 ページ目から始める。送信された
	// フォームは関連ページのページネーション状態を持たないためである。
	linkState := viewmodel.PageLinkState{Context: viewmodel.PageLinkContextEdit}.Normalized()

	// リンクデータを取得
	linkData, err := h.getEditLinkDataUC.Execute(ctx, usecase.GetEditLinkDataInput{
		Page:                   output.Page,
		DraftPage:              output.DraftPage,
		SpaceID:                output.Space.ID,
		CurrentPage:            linkState.LinkPage,
		LinkLimit:              viewmodel.LinkLimit,
		BacklinkLimit:          viewmodel.BacklinkLimit,
		PageBacklinkLimit:      viewmodel.PageBacklinkLimit,
		LinkedPageNumber:       linkState.LinkedPageNumber,
		LinkedPageBacklinkPage: linkState.LinkedBacklinkPage,
		PageBacklinkPage:       linkState.PageBacklinkPage,
	})
	if err != nil {
		slog.ErrorContext(ctx, "リンクデータの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	linkResult := buildEditLinkResult(buildEditLinkResultInput{
		LinkData:        linkData,
		SpaceIdentifier: output.Space.Identifier,
		Page:            output.Page,
		State:           linkState,
	})

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "page_edit_title", map[string]any{
		"SpaceName": output.Space.Name,
	})
	meta.CurrentSpaceIdentifier = spaceIdentVM

	content := pagepages.Edit(pagepages.EditPageData{
		CSRFToken:        csrfToken,
		FormErrors:       formErrors,
		Page:             pageVM,
		Space:            spaceVM,
		Topic:            topicVM,
		LinkList:         linkResult.LinkList,
		BacklinkList:     linkResult.BacklinkList,
		RelatedPageState: linkState,
		ManualSaveURL:    string(templates.PageDraftPagePath(spaceIdentVM, int32(output.Page.Number))),
		// The editor stays within a single space, so omit the space label on each draft card.
		// [Ja] 編集画面は同一スペース内のため、各下書きカードのスペースラベルを省く。
		DraftPages: viewmodel.NewCardLinkDraftPagesWithoutSpace(output.DraftPages),
		ZenMode:    zenModeFromRequest(r),
	})

	currentUser := middleware.UserFromContext(ctx)

	// The editor supplies the global-nav state via GlobalNav. PageNamePageEdit matches no nav item,
	// so no item is highlighted (the draft list column, not the nav, handles in-screen navigation).
	//
	// [Ja] 編集画面はグローバルナビの状態を GlobalNav で供給する。PageNamePageEdit はどのナビ項目にも
	// 一致しないため、いずれの項目もアクティブにならない (画面内のナビゲーションはナビではなく
	// 下書き一覧カラムが担う)。
	navData := components.GlobalNavData{
		CurrentPageName: templates.PageNamePageEdit,
		SignedIn:        currentUser != nil,
		SpaceIdentifier: spaceIdentVM,
	}
	if currentUser != nil {
		navData.UserAtname = currentUser.Atname
	}

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		HideFooter: true,
		GlobalNav:  navData,

		BreadcrumbHeader: pageBreadcrumbHeaderData(ctx, spaceVM, topicVM, editBreadcrumbMaxWidthClass, true),
	}

	w.WriteHeader(http.StatusUnprocessableEntity)

	err = layouts.Default(layoutData, content).Render(ctx, w)
	if err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
	}
}

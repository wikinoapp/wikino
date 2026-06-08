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

		h.renderEditWithErrors(w, r, spaceIdentifier, output, title, body, ve)
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

// renderEditWithErrors はバリデーションエラー時に編集画面を再表示します
func (h *Handler) renderEditWithErrors(
	w http.ResponseWriter,
	r *http.Request,
	spaceIdentifier model.SpaceIdentifier,
	output *usecase.GetPageDetailOutput,
	title string,
	body string,
	formErrors *model.ValidationError,
) {
	ctx := r.Context()

	spaceIdentVM := viewmodel.NewSpaceIdentifier(spaceIdentifier)

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
	meta.SetTitleWithoutSuffix(ctx, "page_edit_title", map[string]any{
		"SpaceName": output.Space.Name,
	})
	meta.CurrentSpaceIdentifier = spaceIdentVM

	content := pagepages.Edit(pagepages.EditPageData{
		CSRFToken:     csrfToken,
		FormErrors:    formErrors,
		Page:          pageVM,
		Space:         spaceVM,
		Topic:         topicVM,
		LinkList:      linkResult.LinkList,
		BacklinkList:  linkResult.BacklinkList,
		ManualSaveURL: string(templates.PageDraftPagePath(spaceIdentVM, int32(output.Page.Number))),
		// The editor stays within a single space, so omit the space label on each draft card.
		// [Ja] 編集画面は同一スペース内のため、各下書きカードのスペースラベルを省く。
		DraftPages: viewmodel.NewCardLinkDraftPagesWithoutSpace(output.DraftPages),
	})

	currentUser := middleware.UserFromContext(ctx)

	// The page editor hides the global sidebar: the left draft list column takes over in-screen
	// draft navigation, so HideSidebar is set and no Sidebar data is built.
	// [Ja] ページ編集画面はグローバルサイドバーを非表示にする。左カラムの下書き一覧が画面内の下書き
	// ナビゲーションを担うため、HideSidebar を立て、Sidebar データは構築しない。
	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		HideFooter:  true,
		HideSidebar: true,
		BottomNav: components.BottomNavData{
			CurrentPageName: templates.PageNamePageEdit,
			SignedIn:        currentUser != nil,
			SpaceIdentifier: spaceIdentVM,
		},
	}

	w.WriteHeader(http.StatusUnprocessableEntity)

	err = layouts.Default(layoutData, content).Render(ctx, w)
	if err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
	}
}

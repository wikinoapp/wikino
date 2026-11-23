package page

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/httperror"
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

// Updateはページを公開します (PATCH /s/{space_identifier}/pages/{page_number})
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
		httperror.NotFound(w, r)
		return
	}

	// フォームデータを取得
	form := publishForm{
		title: r.FormValue("title"),
		body:  r.FormValue("body"),
	}

	// 編集画面が公開の前提にした下書きの版。前提の分からない公開は別の場所の変更を上書きしうるため、
	// 公開せずに、競合したときと同じく送信された内容を残した編集画面で上書きするかを選ばせる
	draftPageDigests, ok := r.Form["draft_page_digest"]
	if !ok || len(draftPageDigests) == 0 {
		h.renderEditForConflict(w, r, user, spaceIdentifier, int32(pageNumber), form)
		return
	}
	form.draftPageDigest = draftPageDigests[0]

	// UseCaseを実行
	publishOutput, err := h.publishPageUC.Execute(ctx, usecase.PublishPageInput{
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pageNumber),
		UserID:          user.ID,
		Title:           form.title,
		Body:            form.body,
		DraftPageDigest: form.draftPageDigest,
	})
	if err != nil {
		h.handleUpdateError(w, r, err, user, spaceIdentifier, int32(pageNumber), form)
		return
	}

	// フラッシュメッセージを設定してリダイレクト
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_page_saved"))
	pagePath := string(templates.PagePath(viewmodel.NewSpaceIdentifier(spaceIdentifier), viewmodel.PageNumber(publishOutput.Page.Number)))
	http.Redirect(w, r, pagePath, http.StatusSeeOther)
}

// publishFormは公開のフォームで送信された値。編集画面を再表示するときに画面へ戻す
type publishForm struct {
	title string
	body  string
	// draftPageDigestは、編集画面が公開の前提にした下書きの版
	draftPageDigest string
}

func (h *Handler) handleUpdateError(w http.ResponseWriter, r *http.Request, err error, user *model.User, spaceIdentifier model.SpaceIdentifier, pageNumber int32, form publishForm) {
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

		h.renderEdit(w, r, output, form, ve, nil)
		return
	}

	if ae := model.AsAppError(err); ae != nil {
		switch ae.Code {
		case model.AppErrCodeResourceNotFound, model.AppErrCodeForbidden:
			httperror.NotFound(w, r)
		case model.AppErrCodeConflict:
			// 下書きが、編集画面を開いたあとに別の場所で更新されていた
			h.renderEditForConflict(w, r, user, spaceIdentifier, pageNumber, form)
		default:
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	slog.ErrorContext(ctx, "ページの公開に失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// renderEditForConflictは、下書きの版が公開の前提と食い違ったときに、送信されたタイトルと本文を残した
// 編集画面を競合の知らせ付きで再表示します。利用者は最新の内容を読み込むか、今の版を前提に公開し直すかを選びます。
func (h *Handler) renderEditForConflict(w http.ResponseWriter, r *http.Request, user *model.User, spaceIdentifier model.SpaceIdentifier, pageNumber int32, form publishForm) {
	ctx := r.Context()

	output, err := h.getPageDetailUC.Execute(ctx, usecase.GetPageDetailInput{
		SpaceIdentifier:   spaceIdentifier,
		PageNumber:        pageNumber,
		UserID:            user.ID,
		IncludeDraftPages: true,
	})
	if err != nil {
		slog.ErrorContext(ctx, "競合の再表示用データの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	// 版を送らない公開はUseCaseの認可を経ていないため、編集画面の表示と同じく権限を確かめる
	if output == nil || !output.CanUpdatePage {
		httperror.NotFound(w, r)
		return
	}

	var currentDigest string
	if output.DraftPage != nil {
		currentDigest = output.DraftPage.ContentDigest()
	}
	h.renderEdit(w, r, output, form, nil, &pagepages.PublishConflict{CurrentDraftPageDigest: currentDigest})
}

// renderEditは公開できなかったときに、送信されたタイトルと本文を残した編集画面を再表示します。
// 競合 (publishConflictがnilでない) なら409、バリデーションエラーなら422で返します。
// スペース識別子を含むリンクの組み立てにはoutputに含まれる保存済みの値を使うため、
// URLパラメータ由来の識別子は受け取りません。
func (h *Handler) renderEdit(
	w http.ResponseWriter,
	r *http.Request,
	output *usecase.GetPageDetailOutput,
	form publishForm,
	formErrors *model.ValidationError,
	publishConflict *pagepages.PublishConflict,
) {
	ctx := r.Context()

	// ViewModelを生成。公開は下書きを変えずに失敗したため、再表示した画面も送信された版を前提にし続ける。
	// 版を送らない画面から公開したときは前提が分からないため、競合の知らせから選ばせる
	pageVM := viewmodel.NewPageFromFormInput(form.title, form.body, output.Page.Number, form.draftPageDigest)
	spaceVM := viewmodel.NewSpace(output.Space)
	topicVM := viewmodel.NewTopic(output.Topic)

	// URLではなく保存済みの識別子からリンクを組み立て、画面内のリンクの表記を揃える。
	spaceIdentVM := spaceVM.Identifier

	// バリデーションエラー後の編集画面の再描画では、各一覧を1ページ目から始める。送信された
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
		"TopicName": output.Topic.Name,
		"SpaceName": output.Space.Name,
	})
	meta.CurrentSpaceIdentifier = spaceIdentVM
	meta.FeatureScript = viewmodel.FeatureScriptEditor

	content := pagepages.Edit(pagepages.EditPageData{
		CSRFToken:        csrfToken,
		FormErrors:       formErrors,
		Page:             pageVM,
		Space:            spaceVM,
		Topic:            topicVM,
		LinkList:         linkResult.LinkList,
		BacklinkList:     linkResult.BacklinkList,
		RelatedPageState: linkState,
		// 編集画面は同一スペース内のため、各下書きカードのスペースラベルを省く。
		DraftPages:      viewmodel.NewCardLinkDraftPagesWithoutSpace(output.DraftPages),
		ZenMode:         zenModeFromRequest(r),
		PublishConflict: publishConflict,
	})

	currentUser := middleware.UserFromContext(ctx)

	// 編集画面はグローバルナビの状態をGlobalNavで供給する。PageNamePageEditはどのナビ項目にも
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

		BreadcrumbHeader: editBreadcrumbHeaderData(ctx, spaceVM, topicVM),
	}

	status := http.StatusUnprocessableEntity
	if publishConflict != nil {
		status = http.StatusConflict
	}
	w.WriteHeader(status)

	err = layouts.Default(layoutData, content).Render(ctx, w)
	if err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
	}
}

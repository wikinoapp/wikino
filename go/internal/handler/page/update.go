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
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	pagepages "github.com/wikinoapp/wikino/go/internal/templates/pages/page"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Update はページを公開します (PATCH /go/s/{space_identifier}/pages/{page_number})
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 認証済みユーザーを取得
	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	// URLパラメータを取得
	spaceIdentifier := chi.URLParam(r, "space_identifier")
	pageNumberStr := chi.URLParam(r, "page_number")

	pageNumber, err := strconv.ParseInt(pageNumberStr, 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// スペースを取得
	space, err := h.spaceRepo.FindByIdentifier(ctx, spaceIdentifier)
	if err != nil {
		slog.ErrorContext(ctx, "スペースの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if space == nil {
		http.NotFound(w, r)
		return
	}

	// スペースメンバーを取得
	spaceMember, err := h.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, model.UserID(user.ID))
	if err != nil {
		slog.ErrorContext(ctx, "スペースメンバーの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if spaceMember == nil {
		http.NotFound(w, r)
		return
	}

	// ページを取得
	pg, err := h.pageRepo.FindBySpaceAndNumber(ctx, space.ID, int32(pageNumber))
	if err != nil {
		slog.ErrorContext(ctx, "ページの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if pg == nil {
		http.NotFound(w, r)
		return
	}

	// トピックメンバーを取得してTopicPolicyを生成
	topicMember, err := h.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, pg.TopicID)
	if err != nil {
		slog.ErrorContext(ctx, "トピックメンバーの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	topicPolicy := policy.NewTopicPolicy(spaceMember.Role, space.ID, topicMember, spaceMember.Active)
	if !topicPolicy.CanUpdatePage(pg) {
		http.NotFound(w, r)
		return
	}

	// フォームデータを取得
	title := r.FormValue("title")
	body := r.FormValue("body")

	// バリデーション
	validator := NewUpdateValidator(h.pageRepo)
	validationResult := validator.Validate(ctx, UpdateValidatorInput{
		Title:           title,
		PageID:          pg.ID,
		TopicID:         pg.TopicID,
		SpaceID:         space.ID,
		SpaceIdentifier: spaceIdentifier,
	})

	if validationResult.FormErrors.HasErrors() {
		h.renderEditWithErrors(w, r, spaceIdentifier, space, pg, title, body, validationResult.FormErrors)
		return
	}

	// トピックを取得（UseCase入力用）
	topic, err := h.topicRepo.FindBySpaceAndID(ctx, space.ID, pg.TopicID)
	if err != nil {
		slog.ErrorContext(ctx, "トピックの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if topic == nil {
		slog.ErrorContext(ctx, "ページのトピックが見つかりません", "page_id", pg.ID, "topic_id", pg.TopicID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// DraftPageを取得（存在する場合はIDが必要）
	draftPage, err := h.draftPageRepo.FindByPageAndMember(ctx, pg.ID, spaceMember.ID, space.ID)
	if err != nil {
		slog.ErrorContext(ctx, "下書きの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Titleをポインタに変換（空文字列の場合はnil）
	var titlePtr *string
	if title != "" {
		titlePtr = &title
	}

	// ページ公開ユースケースを実行
	input := usecase.PublishPageInput{
		SpaceID:          space.ID,
		PageID:           pg.ID,
		SpaceMemberID:    spaceMember.ID,
		TopicID:          pg.TopicID,
		Title:            titlePtr,
		Body:             body,
		SpaceIdentifier:  spaceIdentifier,
		CurrentTopicName: topic.Name,
	}

	// DraftPageが存在する場合はDraftPageIDを設定
	if draftPage != nil {
		input.DraftPageID = draftPage.ID
	}

	_, err = h.publishPageUC.Execute(ctx, input)
	if err != nil {
		slog.ErrorContext(ctx, "ページの公開に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// フラッシュメッセージを設定してリダイレクト
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_page_saved"))
	pagePath := fmt.Sprintf("/s/%s/pages/%d", spaceIdentifier, pg.Number)
	http.Redirect(w, r, pagePath, http.StatusSeeOther)
}

// renderEditWithErrors はバリデーションエラー時に編集画面を再表示します
func (h *Handler) renderEditWithErrors(
	w http.ResponseWriter,
	r *http.Request,
	spaceIdentifier string,
	space *model.Space,
	pg *model.Page,
	title string,
	body string,
	formErrors *session.FormErrors,
) {
	ctx := r.Context()

	// トピックを取得（パンくずリスト用）
	topic, err := h.topicRepo.FindBySpaceAndID(ctx, space.ID, pg.TopicID)
	if err != nil {
		slog.ErrorContext(ctx, "トピックの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if topic == nil {
		slog.ErrorContext(ctx, "ページのトピックが見つかりません", "page_id", pg.ID, "topic_id", pg.TopicID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// ViewModelを生成
	pageVM := viewmodel.NewPageFromFormInput(title, body, pg.Number)
	spaceVM := viewmodel.NewSpace(space)
	topicVM := viewmodel.NewTopic(topic)

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "page_edit_title")

	content := pagepages.Edit(pagepages.EditPageData{
		CSRFToken:  csrfToken,
		FormErrors: formErrors,
		Page:       pageVM,
		Space:      spaceVM,
		Topic:      topicVM,
	})

	layoutData := layouts.DefaultLayoutData{
		Meta:                 meta,
		HideFooter:           true,
		DefaultSidebarClosed: true,
	}

	w.WriteHeader(http.StatusUnprocessableEntity)

	err = layouts.Default(layoutData, content).Render(ctx, w)
	if err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
	}
}

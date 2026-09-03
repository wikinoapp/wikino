package page_move

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
	pagemovepages "github.com/wikinoapp/wikino/go/internal/templates/pages/page_move"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// New はページ移動フォームを表示します (GET /s/{space_identifier}/pages/{page_number}/move)
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
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

	// UseCaseでデータを取得（認可チェック含む）
	output, err := h.getPageMoveDataUC.Execute(ctx, usecase.GetPageMoveDataInput{
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pageNumber),
		UserID:          user.ID,
	})
	if err != nil {
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
		slog.ErrorContext(ctx, "ページ移動データの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.renderMoveForm(w, r, user, spaceIdentifier, output, nil)
}

// renderMoveForm はページ移動フォームをレンダリングします
func (h *Handler) renderMoveForm(
	w http.ResponseWriter,
	r *http.Request,
	user *model.User,
	spaceIdentifier model.SpaceIdentifier,
	output *usecase.GetPageMoveDataOutput,
	formErrors *model.ValidationError,
) {
	ctx := r.Context()

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	spaceIdentVM := viewmodel.NewSpaceIdentifier(spaceIdentifier)

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "page_move_title", map[string]any{
		"SpaceName": output.Space.Name,
	})
	meta.CurrentSpaceIdentifier = spaceIdentVM

	pageVM := viewmodel.NewPageForMove(output.Page)
	spaceVM := viewmodel.NewSpace(output.Space)
	currentTopicVM := viewmodel.NewTopic(output.CurrentTopic)

	// 移動先トピック一覧のViewModel
	topicsForSelect := make([]viewmodel.TopicForSelect, len(output.AvailableTopics))
	for i, t := range output.AvailableTopics {
		topicsForSelect[i] = viewmodel.NewTopicForSelect(t)
	}

	content := pagemovepages.New(pagemovepages.MovePageData{
		CSRFToken:       csrfToken,
		FormErrors:      formErrors,
		Page:            pageVM,
		Space:           spaceVM,
		CurrentTopic:    currentTopicVM,
		AvailableTopics: topicsForSelect,
	})

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNamePageMove,
			SignedIn:        true,
			UserAtname:      user.Atname,
			SpaceIdentifier: spaceIdentVM,
		},

		BreadcrumbHeader: components.BreadcrumbHeaderData{
			MaxWidthClass: "max-w-2xl",
			Items: []components.BreadcrumbItem{
				{
					Path:      templates.HomePath(),
					IconName:  "house-regular",
					AriaLabel: i18n.T(ctx, "breadcrumb_home"),
				},
				{
					Label: spaceVM.Name,
					Path:  templates.SpacePath(spaceIdentVM),
				},
				{
					Label:    currentTopicVM.Name,
					Path:     templates.TopicPath(spaceIdentVM, currentTopicVM.Number),
					IconName: currentTopicVM.IconName,
				},
			},
		},
	}

	err := layouts.Default(layoutData, content).Render(ctx, w)
	if err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

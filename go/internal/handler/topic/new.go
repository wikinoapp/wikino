package topic

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	topicpages "github.com/wikinoapp/wikino/go/internal/templates/pages/topic"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Newはトピック作成フォームを表示します (GET /s/{space_identifier}/topics/new)
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	output, err := h.getTopicNewUsecase.Execute(ctx, usecase.GetTopicNewInput{
		SpaceIdentifier: spaceIdentifier,
		UserID:          user.ID,
	})
	if err != nil {
		h.handleTopicError(w, r, err, "トピック作成フォームのデータの取得に失敗")
		return
	}

	h.renderNewForm(w, r, user, topicpages.NewData{
		CSRFToken: middleware.GetCSRFTokenFromContext(ctx),
		Space:     viewmodel.NewSpace(output.Space),
	})
}

// renderNewFormはトピック作成フォームを描画します。画面のリンクとメタ情報の組み立てに使う
// スペースはデータに含まれており、URLが運んだ識別子ではなく保存済みの識別子から組み立てます。
func (h *Handler) renderNewForm(w http.ResponseWriter, r *http.Request, user *model.User, data topicpages.NewData) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "topic_new_title", map[string]any{"SpaceName": data.Space.Name})
	meta.CurrentSpaceIdentifier = data.Space.Identifier

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameTopicNew,
			SignedIn:        true,
			UserAtname:      user.Atname,
			SpaceIdentifier: data.Space.Identifier,
		},

		BreadcrumbHeader: components.BreadcrumbHeaderData{
			MaxWidthClass: "max-w-2xl",
			Items: append(components.HomeBreadcrumbItems(ctx, true),
				components.BreadcrumbItem{
					Label: data.Space.Name,
					Path:  templates.SpacePath(data.Space.Identifier),
				},
				// この画面が現在地のため、経路はaria-currentを持つリンク無しの項目で締める。
				// ラベルは見出しと同じ文字列だが、パンくずが見出しより短い語を必要としたときに後から
				// 変えられるよう、独立したキーから引く。
				components.BreadcrumbItem{
					Label:     i18n.T(ctx, "topic_new_breadcrumb"),
					IsCurrent: true,
				},
			),
		},
	}

	if err := layouts.Default(layoutData, topicpages.New(data)).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleTopicErrorはトピックのユースケースから返ったエラーを、それに応じたレスポンスへ
// 変える。閲覧者がトピックを作成できないスペースは「見つからない」として答え、そのスペースに誰が
// 参加しているかを外部から試して確かめられないようにする。
func (h *Handler) handleTopicError(w http.ResponseWriter, r *http.Request, err error, logMsg string) {
	ctx := r.Context()

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

	slog.ErrorContext(ctx, logMsg, "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

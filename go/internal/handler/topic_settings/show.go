package topic_settings

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
	topicpages "github.com/wikinoapp/wikino/go/internal/templates/pages/topic"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Showはトピックの設定のトップを表示します
// (GET /s/{space_identifier}/topics/{topic_number}/settings)。
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	// 数値でないトピック番号はどのトピックも指さないため、取得を試みずに「見つからない」として答える。
	topicNumber, err := strconv.ParseInt(chi.URLParam(r, "topic_number"), 10, 32)
	if err != nil {
		httperror.NotFound(w, r)
		return
	}

	output, err := h.getTopicSettingsUC.Execute(ctx, usecase.GetTopicSettingsInput{
		SpaceIdentifier: spaceIdentifier,
		TopicNumber:     int32(topicNumber),
		UserID:          user.ID,
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	spaceVM := viewmodel.NewSpace(output.Space)
	topicVM := viewmodel.NewTopic(output.Topic)

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "topic_settings_title", map[string]any{
		"TopicName": topicVM.Name,
		"SpaceName": spaceVM.Name,
	})
	meta.CurrentSpaceIdentifier = spaceVM.Identifier

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameTopicSettings,
			SignedIn:        true,
			UserAtname:      user.Atname,
			SpaceIdentifier: spaceVM.Identifier,
		},

		BreadcrumbHeader: components.BreadcrumbHeaderData{
			MaxWidthClass: "max-w-2xl",
			Items: append(components.HomeBreadcrumbItems(ctx, true),
				components.BreadcrumbItem{
					Label: spaceVM.Name,
					Path:  templates.SpacePath(spaceVM.Identifier),
				},
				components.BreadcrumbItem{
					Label:    topicVM.Name,
					IconName: topicVM.IconName,
					Path:     templates.TopicPath(spaceVM.Identifier, topicVM.Number),
				},
				// この画面が現在地のため、経路はaria-currentを持つリンク無しの項目で締める。
				components.BreadcrumbItem{
					Label:     i18n.T(ctx, "topic_settings_breadcrumb"),
					IsCurrent: true,
				},
			),
		},
	}

	content := topicpages.Settings(topicpages.SettingsData{
		Space:          spaceVM,
		Topic:          topicVM,
		CanUpdateTopic: output.Items.CanUpdateTopic,
		CanDeleteTopic: output.Items.CanDeleteTopic,
	})

	// ログインした閲覧者だけが開く設定の画面のため、検索エンジンに載せない。
	w.Header().Set("X-Robots-Tag", "noindex")
	if err := layouts.Default(layoutData, content).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleErrorはユースケースから返ったエラーを、それに応じたレスポンスへ変えます。
// 閲覧者が開けないトピックの設定は「見つからない」として答え、そのトピックで誰が何を変更できるかを
// 外部から試して確かめられないようにします。Rails版も同じく404を返していました。
func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	ctx := r.Context()

	if ae := model.AsAppError(err); ae != nil {
		switch ae.Code {
		case model.AppErrCodeResourceNotFound, model.AppErrCodeForbidden:
			httperror.NotFound(w, r)
		default:
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	slog.ErrorContext(ctx, "トピックの設定のデータの取得に失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

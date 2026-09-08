// Package topic_settings_general provides the HTTP handlers of the general settings of a topic.
//
// [Ja] topic_settings_general パッケージはトピックの一般設定の HTTP ハンドラーを提供します。
package topic_settings_general

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	topicpages "github.com/wikinoapp/wikino/go/internal/templates/pages/topic"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Handler serves the screen the general settings of a topic are edited on, and the save it
// submits to.
//
// [Ja] Handler はトピックの一般設定を編集する画面と、その送信先の保存処理を提供します。
type Handler struct {
	cfg                       *config.Config
	flashMgr                  *session.FlashManager
	getTopicSettingsGeneralUC *usecase.GetTopicSettingsGeneralUsecase
	updateTopicUC             *usecase.UpdateTopicUsecase
}

// NewHandler はトピックの一般設定のハンドラーを生成します
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	getTopicSettingsGeneralUC *usecase.GetTopicSettingsGeneralUsecase,
	updateTopicUC *usecase.UpdateTopicUsecase,
) *Handler {
	return &Handler{
		cfg:                       cfg,
		flashMgr:                  flashMgr,
		getTopicSettingsGeneralUC: getTopicSettingsGeneralUC,
		updateTopicUC:             updateTopicUC,
	}
}

// requestTarget reads what the URL names the screen by, and reports whether it could be read. A
// topic number that is not a number belongs to no topic, so it is answered as not found rather
// than looked up.
//
// [Ja] requestTarget は URL が画面を指すために運ぶ値を読み、読めたかどうかを返します。数値でない
// トピック番号はどのトピックも指さないため、取得を試みずに「見つからない」として答えます。
func requestTarget(w http.ResponseWriter, r *http.Request) (model.SpaceIdentifier, int32, *model.User, bool) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return "", 0, nil, false
	}

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	topicNumber, err := strconv.ParseInt(chi.URLParam(r, "topic_number"), 10, 32)
	if err != nil {
		handler.NotFound(w, r)
		return "", 0, nil, false
	}

	return spaceIdentifier, int32(topicNumber), user, true
}

// render draws the general settings screen into the layout it shares with the rest of the topic
// screens. The breadcrumb leads back to the settings of the topic, which stays on the Rails
// version.
//
// [Ja] render は一般設定画面を、トピックの他の画面と共有するレイアウトへ描画します。パンくずは
// トピックの設定へ戻り、その画面は Rails 版のまま残ります。
func (h *Handler) render(w http.ResponseWriter, r *http.Request, user *model.User, data topicpages.SettingsGeneralData) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "topic_settings_general_title", map[string]any{
		"TopicName": data.Topic.Name,
		"SpaceName": data.Space.Name,
	})
	meta.CurrentSpaceIdentifier = data.Space.Identifier

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameTopicSettingsGeneral,
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
				components.BreadcrumbItem{
					Label:    data.Topic.Name,
					IconName: data.Topic.IconName,
					Path:     templates.TopicPath(data.Space.Identifier, data.Topic.Number),
				},
				components.BreadcrumbItem{
					Label: i18n.T(ctx, "topic_settings_general_breadcrumb_settings"),
					Path:  templates.TopicSettingsPath(data.Space.Identifier, data.Topic.Number),
				},
			),
		},
	}

	if err := layouts.Default(layoutData, topicpages.SettingsGeneral(data)).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleError turns an error from a topic usecase into the response it deserves. A topic the
// viewer may not change is answered as not found, so that who may change what in a space is not
// something an outsider can probe for.
//
// [Ja] handleError はトピックのユースケースから返ったエラーを、それに応じたレスポンスへ変えます。
// 閲覧者が変更できないトピックは「見つからない」として答え、そのスペースで誰が何を変更できるかを
// 外部から試して確かめられないようにします。
func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error, logMsg string) {
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

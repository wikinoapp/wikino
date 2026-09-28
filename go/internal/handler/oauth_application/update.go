package oauth_application

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	oauthapppages "github.com/wikinoapp/wikino/go/internal/templates/pages/oauth_application"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// UpdateはOAuthアプリの名前とリダイレクトURIを更新します (PATCH /s/{space_identifier}/settings/oauth_applications/{oauth_application_id})。
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	version, err := strconv.ParseInt(r.PostFormValue("version"), 10, 64)
	if err != nil || version < 1 {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	input := usecase.UpdateOAuthApplicationInput{
		SpaceIdentifier:    model.SpaceIdentifier(chi.URLParam(r, "space_identifier")),
		UserID:             user.ID,
		OAuthApplicationID: model.OAuthApplicationID(chi.URLParam(r, "oauth_application_id")),
		ExpectedVersion:    version,
		Name:               r.PostFormValue("name"),
		RedirectURIs:       r.PostFormValue("redirect_uris"),
	}

	output, err := h.updateOAuthApplicationUC.Execute(ctx, input)
	if err != nil {
		h.handleUpdateError(w, r, err, user, input)
		return
	}

	spaceVM := viewmodel.NewSpace(output.Space)
	appVM := viewmodel.NewOAuthApplication(output.Application, nil)

	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_oauth_application_updated", map[string]any{"Name": appVM.Name}))
	http.Redirect(w, r, string(templates.SpaceSettingsOAuthApplicationPath(spaceVM.Identifier, appVM.ID)), http.StatusSeeOther)
}

// handleUpdateErrorは入力が拒否されたときに送信された内容を戻したフォームを再描画し、
// 競合したときは現在の値のフォームと、保存されなかった送信の値を並べて再描画する。
// それ以外は他の画面と同じ形で応答する。拒否された送信は何も変えていないため、見出しと
// パンくずにはアプリを読み直したものを使う。
func (h *Handler) handleUpdateError(w http.ResponseWriter, r *http.Request, err error, user *model.User, input usecase.UpdateOAuthApplicationInput) {
	ctx := r.Context()

	if ae := model.AsAppError(err); ae != nil && ae.Code == model.AppErrCodeConflict {
		output, getErr := h.getOAuthApplicationEditUC.Execute(ctx, usecase.GetOAuthApplicationEditInput{
			SpaceIdentifier:    input.SpaceIdentifier,
			UserID:             user.ID,
			OAuthApplicationID: input.OAuthApplicationID,
		})
		if getErr != nil {
			h.handleError(w, r, getErr, "競合したOAuthアプリの再取得に失敗")
			return
		}
		ve := model.NewValidationError()
		ve.AddGlobal(ae.UserMsg)
		h.renderEditForm(w, r, user, http.StatusConflict, oauthapppages.EditData{
			CSRFToken:   middleware.GetCSRFTokenFromContext(ctx),
			Space:       viewmodel.NewSpace(output.Space),
			Application: viewmodel.NewOAuthApplication(output.Application, nil),
			Version:     output.Application.Version,
			Fields: oauthapppages.FormFieldsData{
				FormErrors:   ve,
				Name:         output.Application.Name,
				RedirectURIs: strings.Join(output.Application.RedirectURIs, "\n"),
			},
			Conflicted: &oauthapppages.ConflictedFieldsData{
				Name:         input.Name,
				RedirectURIs: input.RedirectURIs,
			},
		})
		return
	}

	ve := model.AsValidationError(err)
	if ve == nil {
		h.handleError(w, r, err, "OAuthアプリの更新に失敗")
		return
	}

	output, getErr := h.getOAuthApplicationEditUC.Execute(ctx, usecase.GetOAuthApplicationEditInput{
		SpaceIdentifier:    input.SpaceIdentifier,
		UserID:             user.ID,
		OAuthApplicationID: input.OAuthApplicationID,
	})
	if getErr != nil {
		h.handleError(w, r, getErr, "フォーム再表示用データの取得に失敗")
		return
	}

	h.renderEditForm(w, r, user, http.StatusUnprocessableEntity, oauthapppages.EditData{
		CSRFToken:   middleware.GetCSRFTokenFromContext(ctx),
		Space:       viewmodel.NewSpace(output.Space),
		Application: viewmodel.NewOAuthApplication(output.Application, nil),
		Version:     input.ExpectedVersion,
		Fields: oauthapppages.FormFieldsData{
			FormErrors:   ve,
			Name:         input.Name,
			RedirectURIs: input.RedirectURIs,
		},
	})
}

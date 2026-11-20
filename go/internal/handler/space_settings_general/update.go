package space_settings_general

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	spacepages "github.com/wikinoapp/wikino/go/internal/templates/pages/space"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Updateはスペースの一般設定を保存します (PATCH /s/{space_identifier}/settings/general)。
// 識別子を変えるとスペースのURLが変わるため、保存できたら新しい識別子の一般設定へリダイレクトします。
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	spaceIdentifier, user, ok := requestTarget(w, r)
	if !ok {
		return
	}

	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	input := usecase.UpdateSpaceInput{
		SpaceIdentifier: spaceIdentifier,
		UserID:          user.ID,
		Identifier:      r.FormValue("identifier"),
		Name:            r.FormValue("name"),
	}

	output, err := h.updateSpaceUC.Execute(ctx, input)
	if err != nil {
		h.handleUpdateError(w, r, err, user, input)
		return
	}

	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_space_updated"))
	http.Redirect(w, r, string(templates.SpaceSettingsGeneralPath(viewmodel.NewSpaceIdentifier(output.Space.Identifier))), http.StatusSeeOther)
}

// handleUpdateErrorは入力が拒否されたときに送信された内容を戻したフォームを再描画し、
// それ以外は一般設定の画面と同じ形で応答します。拒否された送信は何も変えていないため、
// パンくずとフォームの送信先には、スペースを読み直したもの (保存されている識別子) を使います。
func (h *Handler) handleUpdateError(w http.ResponseWriter, r *http.Request, err error, user *model.User, input usecase.UpdateSpaceInput) {
	ctx := r.Context()

	ve := model.AsValidationError(err)
	if ve == nil {
		h.handleError(w, r, err, "スペースの一般設定の更新に失敗")
		return
	}

	output, getErr := h.getSpaceSettingsGeneralUC.Execute(ctx, usecase.GetSpaceSettingsGeneralInput{
		SpaceIdentifier: input.SpaceIdentifier,
		UserID:          user.ID,
	})
	if getErr != nil {
		h.handleError(w, r, getErr, "フォーム再表示用データの取得に失敗")
		return
	}

	w.WriteHeader(http.StatusUnprocessableEntity)
	h.render(w, r, user, spacepages.SettingsGeneralData{
		CSRFToken: middleware.GetCSRFTokenFromContext(ctx),
		Space:     viewmodel.NewSpace(output.Space),
		Fields: spacepages.FormFieldsData{
			FormErrors: ve,
			Identifier: input.Identifier,
			Name:       input.Name,
		},
	})
}

package profile

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/httperror"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	profilepages "github.com/wikinoapp/wikino/go/internal/templates/pages/profile"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Showはプロフィールの画面を表示します (GET /@{atname})。ログインしていなくても開けます。
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	var viewerUserID *model.UserID
	if user != nil {
		viewerUserID = &user.ID
	}

	output, err := h.getProfileShowUC.Execute(ctx, usecase.GetProfileShowInput{
		Atname:       chi.URLParam(r, "atname"),
		ViewerUserID: viewerUserID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "プロフィールの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if output == nil {
		httperror.NotFound(w, r)
		return
	}

	profileVM := viewmodel.NewProfile(output.User)

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	if profileVM.Name == "" {
		meta.SetTitleWithoutSuffix(ctx, "profile_show_title_without_name", map[string]any{"Atname": profileVM.Atname})
	} else {
		meta.SetTitleWithoutSuffix(ctx, "profile_show_title", map[string]any{"Name": profileVM.Name, "Atname": profileVM.Atname})
	}
	meta.OGURL = h.cfg.AppURL() + string(templates.ProfilePath(profileVM.Atname))
	// 自己紹介があれば、プロフィールごとの要約として検索結果やリンクのプレビューに出す
	if profileVM.Description != "" {
		meta.Description = profileVM.Description
	}

	signedIn := user != nil
	var userAtname string
	if user != nil {
		userAtname = user.Atname
	}

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameProfile,
			SignedIn:        signedIn,
			UserAtname:      userAtname,
		},

		BreadcrumbHeader: components.BreadcrumbHeaderData{
			MaxWidthClass: "max-w-3xl",
		},
	}

	content := profilepages.Show(profilepages.ShowPageData{
		Profile:        profileVM,
		JoinedSpaces:   viewmodel.NewSpaces(output.JoinedSpaces),
		CanEditProfile: output.CanEditProfile,
	})

	err = layouts.Default(layoutData, content).Render(ctx, w)
	if err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

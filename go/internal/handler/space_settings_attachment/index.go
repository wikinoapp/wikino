package space_settings_attachment

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/httperror"
	"github.com/wikinoapp/wikino/go/internal/httppagination"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	spacepages "github.com/wikinoapp/wikino/go/internal/templates/pages/space"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// attachmentIndexPageLimitは添付ファイルの管理の画面の1ページあたりの表示件数です (Rails版と同じ)。
const attachmentIndexPageLimit = 50

// Indexはスペースの添付ファイルを新しい順に表示します (GET /s/{space_identifier}/settings/attachments)。
// `q` があればファイル名で絞り込みます。
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	spaceIdentifier, user, ok := requestTarget(w, r)
	if !ok {
		return
	}

	// SQLのoffsetがクエリのint32パラメーターに収まらないページは、UseCaseを呼ぶ前に拒否する。
	currentPage, ok := httppagination.ParsePageParam(r, attachmentIndexPageLimit)
	if !ok {
		httperror.NotFound(w, r)
		return
	}
	searchQuery := r.URL.Query().Get("q")

	output, err := h.getSpaceSettingsAttachmentsUC.Execute(ctx, usecase.GetSpaceSettingsAttachmentsInput{
		SpaceIdentifier: spaceIdentifier,
		UserID:          user.ID,
		Query:           searchQuery,
		Page:            currentPage,
		PageLimit:       attachmentIndexPageLimit,
	})
	if err != nil {
		h.handleError(w, r, err, "添付ファイルの管理の画面のデータの取得に失敗")
		return
	}

	pagination := viewmodel.NewPagination(int(currentPage), output.TotalCount, attachmentIndexPageLimit)
	if pagination.Current > pagination.Total {
		httperror.NotFound(w, r)
		return
	}

	spaceVM := viewmodel.NewSpace(output.Space)

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	titleKey := "space_settings_attachment_index_title"
	if currentPage > 1 {
		titleKey = "space_settings_attachment_index_paginated_title"
	}
	meta.SetTitleWithoutSuffix(ctx, titleKey, map[string]any{
		"SpaceName":  spaceVM.Name,
		"PageNumber": currentPage,
	})
	meta.CurrentSpaceIdentifier = spaceVM.Identifier

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameSpaceSettingsAttachmentIndex,
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
					Label: i18n.T(ctx, "space_settings_attachment_index_breadcrumb_settings"),
					Path:  templates.SpaceSettingsPath(spaceVM.Identifier),
				},
				// この画面が現在地のため、経路はaria-currentを持つリンク無しの項目で締める。
				components.BreadcrumbItem{
					Label:     i18n.T(ctx, "space_settings_attachment_index_breadcrumb"),
					IsCurrent: true,
				},
			),
		},
	}

	content := spacepages.SettingsAttachments(spacepages.SettingsAttachmentsData{
		CSRFToken:           middleware.GetCSRFTokenFromContext(ctx),
		Space:               spaceVM,
		Query:               searchQuery,
		Attachments:         viewmodel.NewSettingsAttachments(output.Attachments, output.ThumbnailURLs),
		Pagination:          pagination,
		CanDeleteAttachment: output.CanDeleteAttachment,
	})

	// ログインしたメンバーだけが開く画面のため、検索エンジンに載せない。
	w.Header().Set("X-Robots-Tag", "noindex")
	// 閲覧者ごとの判定と期限付きのサムネイルの署名付きURLを含むため、ブラウザにも途中のキャッシュにも
	// 保存させない。署名付きURLは資格情報の代わりになる
	w.Header().Set("Cache-Control", "no-store")
	if err := layouts.Default(layoutData, content).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

package draft_page_index

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	draftpagepages "github.com/wikinoapp/wikino/go/internal/templates/pages/draft_page"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Index は下書き一覧画面を表示します (GET /drafts)
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	drafts, err := h.draftPageRepo.ListByUserForIndex(ctx, user.ID)
	if err != nil {
		slog.ErrorContext(ctx, "下書き一覧の取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	groups := viewmodel.NewDraftPageGroupsForIndex(drafts, user.TimeZone)

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "draft_page_index_title")

	flash := h.flashMgr.GetFlash(w, r)

	sidebarContent := h.sidebarHelper.Content(ctx, user.ID)

	layoutData := layouts.DefaultLayoutData{
		Meta:  meta,
		Flash: flash,
		Sidebar: components.SidebarData{
			DefaultClosed:     layouts.SidebarDefaultClosed(r),
			CurrentPageName:   templates.PageNameDraftPageIndex,
			SignedIn:          true,
			UserAtname:        user.Atname,
			JoinedTopics:      sidebarContent.JoinedTopics,
			DraftPages:        sidebarContent.DraftPages,
			HasMoreDraftPages: sidebarContent.HasMoreDraftPages,
		},
	}

	content := draftpagepages.Index(draftpagepages.IndexData{
		Groups: groups,
	})

	err = layouts.Default(layoutData, content).Render(ctx, w)
	if err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

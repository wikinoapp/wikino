package draft_page_revision

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// GetDraftPageRevisionはページについてトークンの持ち主の下書きのリビジョンを1件返す
// (GET /api/v1/spaces/{space_identifier}/pages/{page_number}/draft_page_revisions/{draft_page_revision_id})
func (h *Handler) GetDraftPageRevision(ctx context.Context, request apigen.GetDraftPageRevisionRequestObject) (apigen.GetDraftPageRevisionResponseObject, error) {
	output, err := h.getAPIDraftPageRevisionUC.Execute(ctx, usecase.GetAPIDraftPageRevisionInput{
		Principal:       middleware.APIPrincipalFromContext(ctx),
		SpaceIdentifier: model.SpaceIdentifier(request.SpaceIdentifier),
		PageNumber:      request.PageNumber,
		RevisionID:      model.DraftPageRevisionID(request.DraftPageRevisionId.String()),
	})
	if err != nil {
		return nil, err
	}

	revision, err := toAPIDraftPageRevision(output.Revision)
	if err != nil {
		return nil, err
	}
	return apigen.GetDraftPageRevision200JSONResponse{Body: revision}, nil
}

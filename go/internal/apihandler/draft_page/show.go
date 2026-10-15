package draft_page

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/apietag"
	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// GetDraftPageはページについてトークンの持ち主の下書きを返す
// (GET /api/v1/spaces/{space_identifier}/pages/{page_number}/draft_page)。
// `If-None-Match` が下書きのETagと一致すれば、本文の無い304を返す
func (h *Handler) GetDraftPage(ctx context.Context, request apigen.GetDraftPageRequestObject) (apigen.GetDraftPageResponseObject, error) {
	principal := middleware.APIPrincipalFromContext(ctx)
	output, err := h.getAPIDraftPageUC.Execute(ctx, usecase.GetAPIDraftPageInput{
		Principal:       principal,
		SpaceIdentifier: model.SpaceIdentifier(request.SpaceIdentifier),
		PageNumber:      request.PageNumber,
	})
	if err != nil {
		return nil, err
	}

	etag := `"` + output.DraftPage.ContentDigest() + `"`
	if request.Params.IfNoneMatch != nil && apietag.IfNoneMatchMatches(*request.Params.IfNoneMatch, etag) {
		return apigen.GetDraftPage304Response{Headers: apigen.GetDraftPage304ResponseHeaders{ETag: etag}}, nil
	}

	draft, err := ToAPIDraftPage(h.appURL, principal.Space, output.DraftPage, output.Topic)
	if err != nil {
		return nil, err
	}
	return apigen.GetDraftPage200JSONResponse{Body: draft, Headers: apigen.GetDraftPage200ResponseHeaders{ETag: etag}}, nil
}

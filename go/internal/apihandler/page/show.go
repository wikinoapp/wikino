package page

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/apietag"
	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// GetPageはトークンが属するスペースのページを返す (GET /api/v1/spaces/{space_identifier}/pages/{page_number})。
// `If-None-Match` がページのETagと一致すれば、本文の無い304を返す
func (h *Handler) GetPage(ctx context.Context, request apigen.GetPageRequestObject) (apigen.GetPageResponseObject, error) {
	principal := middleware.APIPrincipalFromContext(ctx)
	output, err := h.getAPIPageUC.Execute(ctx, usecase.GetAPIPageInput{
		Principal:       principal,
		SpaceIdentifier: model.SpaceIdentifier(request.SpaceIdentifier),
		PageNumber:      request.PageNumber,
	})
	if err != nil {
		return nil, err
	}

	etag := `"` + output.Page.ContentDigest() + `"`
	if request.Params.IfNoneMatch != nil && apietag.IfNoneMatchMatches(*request.Params.IfNoneMatch, etag) {
		return apigen.GetPage304Response{Headers: apigen.GetPage304ResponseHeaders{ETag: etag}}, nil
	}

	page, err := ToAPIPage(h.appURL, principal.Space, output.Page, output.Topic)
	if err != nil {
		return nil, err
	}
	return apigen.GetPage200JSONResponse{Body: page, Headers: apigen.GetPage200ResponseHeaders{ETag: etag}}, nil
}

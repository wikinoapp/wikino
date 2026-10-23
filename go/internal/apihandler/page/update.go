package page

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/apietag"
	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// UpdatePageはトークンが属するスペースのページを、JSON Merge Patchで更新する
// (PATCH /api/v1/spaces/{space_identifier}/pages/{page_number})。
// `If-Match` が無ければ428、ページの今のETagと一致しなければ412を返し、
// 更新したページを新しい `ETag` を付けて200で返す
func (h *Handler) UpdatePage(ctx context.Context, request apigen.UpdatePageRequestObject) (apigen.UpdatePageResponseObject, error) {
	if request.Params.IfMatch == nil {
		return nil, apierror.ErrPreconditionRequired
	}
	digests, matchAny := apietag.ParseIfMatch(*request.Params.IfMatch)

	principal := middleware.APIPrincipalFromContext(ctx)
	output, err := h.updateAPIPageUC.Execute(ctx, usecase.UpdateAPIPageInput{
		Principal:       principal,
		SpaceIdentifier: model.SpaceIdentifier(request.SpaceIdentifier),
		PageNumber:      request.PageNumber,
		IfMatchDigests:  digests,
		IfMatchAny:      matchAny,
		Title:           request.Body.Title,
		Body:            request.Body.Body,
	})
	if err != nil {
		return nil, err
	}

	page, err := ToAPIPage(h.appURL, principal.Space, output.Page, output.Topic)
	if err != nil {
		return nil, err
	}
	return apigen.UpdatePage200JSONResponse{
		Body:    page,
		Headers: apigen.UpdatePage200ResponseHeaders{ETag: `"` + output.Page.ContentDigest() + `"`},
	}, nil
}

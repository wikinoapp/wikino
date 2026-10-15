package draft_page_publication

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/apietag"
	"github.com/wikinoapp/wikino/go/internal/apigen"
	apipage "github.com/wikinoapp/wikino/go/internal/apihandler/page"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// CreateDraftPagePublicationはページについてトークンの持ち主の下書きを公開する
// (POST /api/v1/spaces/{space_identifier}/pages/{page_number}/draft_page/publication)。
// `If-Match` が無ければ428、下書きの今のETagと一致しなければ412を返す。
// 公開したページを、ページのURLを示す `Content-Location` とgetPageと同じ `ETag` を付けて200で返す
func (h *Handler) CreateDraftPagePublication(ctx context.Context, request apigen.CreateDraftPagePublicationRequestObject) (apigen.CreateDraftPagePublicationResponseObject, error) {
	if request.Params.IfMatch == nil {
		return nil, apierror.ErrPreconditionRequired
	}
	digests, matchAny := apietag.ParseIfMatch(*request.Params.IfMatch)

	principal := middleware.APIPrincipalFromContext(ctx)
	output, err := h.createAPIDraftPagePublicationUC.Execute(ctx, usecase.CreateAPIDraftPagePublicationInput{
		Principal:       principal,
		SpaceIdentifier: model.SpaceIdentifier(request.SpaceIdentifier),
		PageNumber:      request.PageNumber,
		IfMatchDigests:  digests,
		IfMatchAny:      matchAny,
	})
	if err != nil {
		return nil, err
	}

	page, err := apipage.ToAPIPage(h.appURL, principal.Space, output.Page, output.Topic)
	if err != nil {
		return nil, err
	}
	return apigen.CreateDraftPagePublication200JSONResponse{
		Body: page,
		Headers: apigen.CreateDraftPagePublication200ResponseHeaders{
			ContentLocation: apipage.Path(request.SpaceIdentifier, output.Page.Number),
			ETag:            `"` + output.Page.ContentDigest() + `"`,
		},
	}, nil
}

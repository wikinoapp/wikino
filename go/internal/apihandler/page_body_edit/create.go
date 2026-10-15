package page_body_edit

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/apietag"
	"github.com/wikinoapp/wikino/go/internal/apigen"
	apipage "github.com/wikinoapp/wikino/go/internal/apihandler/page"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// CreatePageBodyEditはトークンが属するスペースのページの本文を検索置換する
// (POST /api/v1/spaces/{space_identifier}/pages/{page_number}/body_edits)。
// `If-Match` は任意で、送られた場合だけページの今のETagと照合し、一致しなければ412を返す。
// 更新したページを、ページのURLを示す `Content-Location` とgetPageと同じ `ETag` を付けて200で返す
func (h *Handler) CreatePageBodyEdit(ctx context.Context, request apigen.CreatePageBodyEditRequestObject) (apigen.CreatePageBodyEditResponseObject, error) {
	var digests []string
	var matchAny bool
	if request.Params.IfMatch != nil {
		digests, matchAny = apietag.ParseIfMatch(*request.Params.IfMatch)
	}

	edits := make([]model.BodyEdit, 0, len(request.Body.Edits))
	for _, edit := range request.Body.Edits {
		edits = append(edits, model.BodyEdit{
			OldStr:     edit.OldStr,
			NewStr:     edit.NewStr,
			ReplaceAll: edit.ReplaceAll != nil && *edit.ReplaceAll,
		})
	}

	principal := middleware.APIPrincipalFromContext(ctx)
	output, err := h.createAPIPageBodyEditUC.Execute(ctx, usecase.CreateAPIPageBodyEditInput{
		Principal:       principal,
		SpaceIdentifier: model.SpaceIdentifier(request.SpaceIdentifier),
		PageNumber:      request.PageNumber,
		HasIfMatch:      request.Params.IfMatch != nil,
		IfMatchDigests:  digests,
		IfMatchAny:      matchAny,
		Edits:           edits,
	})
	if err != nil {
		return nil, err
	}

	page, err := apipage.ToAPIPage(h.appURL, principal.Space, output.Page, output.Topic)
	if err != nil {
		return nil, err
	}
	return apigen.CreatePageBodyEdit200JSONResponse{
		Body: page,
		Headers: apigen.CreatePageBodyEdit200ResponseHeaders{
			ContentLocation: apipage.Path(request.SpaceIdentifier, output.Page.Number),
			ETag:            `"` + output.Page.ContentDigest() + `"`,
		},
	}, nil
}

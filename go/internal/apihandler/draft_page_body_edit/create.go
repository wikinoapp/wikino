package draft_page_body_edit

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/apietag"
	"github.com/wikinoapp/wikino/go/internal/apigen"
	apidraftpage "github.com/wikinoapp/wikino/go/internal/apihandler/draft_page"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// CreateDraftPageBodyEditはページについてトークンの持ち主の下書きの本文を検索置換する
// (POST /api/v1/spaces/{space_identifier}/pages/{page_number}/draft_page/body_edits)。
// `If-Match` は任意で、送られた場合だけ下書きの今のETagと照合し、一致しなければ412を返す。
// 更新した下書きを、下書きのURLを示す `Content-Location` とgetDraftPageと同じ `ETag` を付けて200で返す
func (h *Handler) CreateDraftPageBodyEdit(ctx context.Context, request apigen.CreateDraftPageBodyEditRequestObject) (apigen.CreateDraftPageBodyEditResponseObject, error) {
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
	output, err := h.createAPIDraftPageBodyEditUC.Execute(ctx, usecase.CreateAPIDraftPageBodyEditInput{
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

	draft, err := apidraftpage.ToAPIDraftPage(h.appURL, principal.Space, output.DraftPage, output.Topic)
	if err != nil {
		return nil, err
	}
	return apigen.CreateDraftPageBodyEdit200JSONResponse{
		Body: draft,
		Headers: apigen.CreateDraftPageBodyEdit200ResponseHeaders{
			ContentLocation: apidraftpage.Path(request.SpaceIdentifier, output.DraftPage.Page.Number),
			ETag:            `"` + output.DraftPage.ContentDigest() + `"`,
		},
	}, nil
}

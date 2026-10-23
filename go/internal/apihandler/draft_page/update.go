package draft_page

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/apietag"
	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// UpdateDraftPageはページについてトークンの持ち主の下書きを、JSON Merge Patchで更新する
// (PATCH /api/v1/spaces/{space_identifier}/pages/{page_number}/draft_page)。
// `If-Match` が無ければ428、下書きの今のETagと一致しなければ412を返し、
// 更新した下書きを新しい `ETag` を付けて200で返す
func (h *Handler) UpdateDraftPage(ctx context.Context, request apigen.UpdateDraftPageRequestObject) (apigen.UpdateDraftPageResponseObject, error) {
	if request.Params.IfMatch == nil {
		return nil, apierror.ErrPreconditionRequired
	}
	digests, matchAny := apietag.ParseIfMatch(*request.Params.IfMatch)

	principal := middleware.APIPrincipalFromContext(ctx)
	// JSON Merge Patchの `title` は、省略 (変えない) と `null` (タイトルを外す) を区別する
	input := usecase.UpdateAPIDraftPageInput{
		Principal:       principal,
		SpaceIdentifier: model.SpaceIdentifier(request.SpaceIdentifier),
		PageNumber:      request.PageNumber,
		IfMatchDigests:  digests,
		IfMatchAny:      matchAny,
		UpdateTitle:     request.Body.Title.IsSpecified(),
		Body:            request.Body.Body,
	}
	if title, err := request.Body.Title.Get(); err == nil {
		input.Title = &title
	}

	output, err := h.updateAPIDraftPageUC.Execute(ctx, input)
	if err != nil {
		return nil, err
	}

	draft, err := ToAPIDraftPage(h.appURL, principal.Space, output.DraftPage, output.Topic)
	if err != nil {
		return nil, err
	}
	return apigen.UpdateDraftPage200JSONResponse{
		Body:    draft,
		Headers: apigen.UpdateDraftPage200ResponseHeaders{ETag: `"` + output.DraftPage.ContentDigest() + `"`},
	}, nil
}

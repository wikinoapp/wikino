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

// DeleteDraftPageはページについてトークンの持ち主の下書きを、編集履歴ごと削除する
// (DELETE /api/v1/spaces/{space_identifier}/pages/{page_number}/draft_page)。
// `If-Match` が無ければ428、下書きの今のETagと一致しなければ412を返し、削除できれば204を返す
func (h *Handler) DeleteDraftPage(ctx context.Context, request apigen.DeleteDraftPageRequestObject) (apigen.DeleteDraftPageResponseObject, error) {
	if request.Params.IfMatch == nil {
		return nil, apierror.ErrPreconditionRequired
	}
	digests, matchAny := apietag.ParseIfMatch(*request.Params.IfMatch)

	err := h.deleteAPIDraftPageUC.Execute(ctx, usecase.DeleteAPIDraftPageInput{
		Principal:       middleware.APIPrincipalFromContext(ctx),
		SpaceIdentifier: model.SpaceIdentifier(request.SpaceIdentifier),
		PageNumber:      request.PageNumber,
		IfMatchDigests:  digests,
		IfMatchAny:      matchAny,
	})
	if err != nil {
		return nil, err
	}
	return apigen.DeleteDraftPage204Response{}, nil
}

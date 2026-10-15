package page_link

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	apipage "github.com/wikinoapp/wikino/go/internal/apihandler/page"
	"github.com/wikinoapp/wikino/go/internal/apipagination"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// ListPageLinksはページの保存済みの本文がリンクしているページを更新日時の新しい順に返す
// (GET /api/v1/spaces/{space_identifier}/pages/{page_number}/links)
func (h *Handler) ListPageLinks(ctx context.Context, request apigen.ListPageLinksRequestObject) (apigen.ListPageLinksResponseObject, error) {
	principal := middleware.APIPrincipalFromContext(ctx)
	// カーソルはハンドラーで検証するため、その前にトークンが属するスペースかを確かめ、属さないスペースでは未存在として扱う。
	// UseCase側でも同じくトークンが属するスペースかを検証し、認可をHandlerだけに委ねない。
	if !principal.IsBoundTo(model.SpaceIdentifier(request.SpaceIdentifier)) {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	after, err := apipage.DecodePageCursor(ctx, request.Params.Cursor)
	if err != nil {
		return nil, err
	}

	output, err := h.listAPIPageLinksUC.Execute(ctx, usecase.ListAPIPageLinksInput{
		Principal:       principal,
		SpaceIdentifier: model.SpaceIdentifier(request.SpaceIdentifier),
		PageNumber:      request.PageNumber,
		After:           after,
		Limit:           apipagination.Limit(request.Params.Limit),
	})
	if err != nil {
		return nil, err
	}

	body, link, err := apipage.ToAPIPageList(ctx, h.appURL, principal.Space, output.Pages, output.Topics, output.HasNext)
	if err != nil {
		return nil, err
	}
	return apigen.ListPageLinks200JSONResponse{Body: body, Headers: apigen.ListPageLinks200ResponseHeaders{Link: link}}, nil
}

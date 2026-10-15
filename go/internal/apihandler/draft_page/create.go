package draft_page

import (
	"context"
	"fmt"
	"net/url"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// CreateDraftPageはトークンの持ち主の下書きを作成する (POST /api/v1/spaces/{space_identifier}/draft_pages)。
// 作成した下書きを `Location` と、getDraftPageと同じ `ETag` を付けて201で返す
func (h *Handler) CreateDraftPage(ctx context.Context, request apigen.CreateDraftPageRequestObject) (apigen.CreateDraftPageResponseObject, error) {
	principal := middleware.APIPrincipalFromContext(ctx)
	output, err := h.createAPIDraftPageUC.Execute(ctx, usecase.CreateAPIDraftPageInput{
		Principal:       principal,
		SpaceIdentifier: model.SpaceIdentifier(request.SpaceIdentifier),
		PageNumber:      request.Body.PageNumber,
		TopicNumber:     request.Body.TopicNumber,
		Title:           request.Body.Title,
		Body:            request.Body.Body,
	})
	if err != nil {
		return nil, err
	}

	draft, err := ToAPIDraftPage(h.appURL, principal.Space, output.DraftPage, output.Topic)
	if err != nil {
		return nil, err
	}
	return apigen.CreateDraftPage201JSONResponse{
		Body: draft,
		Headers: apigen.CreateDraftPage201ResponseHeaders{
			ETag:     `"` + output.DraftPage.ContentDigest() + `"`,
			Location: Path(request.SpaceIdentifier, output.DraftPage.Page.Number),
		},
	}, nil
}

// Pathは下書きのAPIのURLを、パスから始まる相対参照で返す。
// OpenAPI記述の `servers` (`/api/v1`) にgetDraftPageのパスを続けたもの
func Path(spaceIdentifier string, number model.PageNumber) string {
	return fmt.Sprintf("/api/v1/spaces/%s/pages/%d/draft_page", url.PathEscape(spaceIdentifier), number)
}

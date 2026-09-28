package page

import (
	"context"
	"fmt"
	"net/url"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// CreatePageはトークンを束縛したスペースにページを作成して公開する (POST /api/v1/spaces/{space_identifier}/pages)。
// 作成したページを `Location` と、getPageと同じ `ETag` を付けて201で返す
func (h *Handler) CreatePage(ctx context.Context, request apigen.CreatePageRequestObject) (apigen.CreatePageResponseObject, error) {
	output, err := h.createAPIPageUC.Execute(ctx, usecase.CreateAPIPageInput{
		Principal:       middleware.APIPrincipalFromContext(ctx),
		SpaceIdentifier: model.SpaceIdentifier(request.SpaceIdentifier),
		TopicNumber:     request.Body.TopicNumber,
		Title:           request.Body.Title,
		Body:            request.Body.Body,
	})
	if err != nil {
		return nil, err
	}

	page, err := toAPIPage(output.Page, output.Topic)
	if err != nil {
		return nil, err
	}
	return apigen.CreatePage201JSONResponse{
		Body: page,
		Headers: apigen.CreatePage201ResponseHeaders{
			ETag:     `"` + output.Page.ContentDigest() + `"`,
			Location: pageLocation(request.SpaceIdentifier, output.Page.Number),
		},
	}, nil
}

// pageLocationはページのAPIのURLを、パスから始まる相対参照で返す。
// OpenAPI記述の `servers` (`/api/v1`) にgetPageのパスを続けたもの
func pageLocation(spaceIdentifier string, number model.PageNumber) string {
	return fmt.Sprintf("/api/v1/spaces/%s/pages/%d", url.PathEscape(spaceIdentifier), number)
}

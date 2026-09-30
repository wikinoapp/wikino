package page

import (
	"context"
	"strings"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// GetPageはトークンを束縛したスペースのページを返す (GET /api/v1/spaces/{space_identifier}/pages/{page_number})。
// `If-None-Match` がページのETagと一致すれば、本文の無い304を返す
func (h *Handler) GetPage(ctx context.Context, request apigen.GetPageRequestObject) (apigen.GetPageResponseObject, error) {
	output, err := h.getAPIPageUC.Execute(ctx, usecase.GetAPIPageInput{
		Principal:       middleware.APIPrincipalFromContext(ctx),
		SpaceIdentifier: model.SpaceIdentifier(request.SpaceIdentifier),
		PageNumber:      request.PageNumber,
	})
	if err != nil {
		return nil, err
	}

	etag := `"` + output.Page.ContentDigest() + `"`
	if request.Params.IfNoneMatch != nil && ifNoneMatchMatches(*request.Params.IfNoneMatch, etag) {
		return apigen.GetPage304Response{Headers: apigen.GetPage304ResponseHeaders{ETag: etag}}, nil
	}

	page, err := toAPIPage(output.Page, output.Topic)
	if err != nil {
		return nil, err
	}
	return apigen.GetPage200JSONResponse{Body: page, Headers: apigen.GetPage200ResponseHeaders{ETag: etag}}, nil
}

// ifNoneMatchMatchesは `If-None-Match` の値がetagに一致するかを返す (RFC 9110 §13.1.2)。
// 値は `*` か、カンマ区切りのエンティティタグの並び。`If-None-Match` は弱い比較で判定するため、
// `W/` の有無は区別しない
func ifNoneMatchMatches(ifNoneMatch, etag string) bool {
	if strings.TrimSpace(ifNoneMatch) == "*" {
		return true
	}
	for tag := range strings.SplitSeq(ifNoneMatch, ",") {
		if strings.TrimPrefix(strings.TrimSpace(tag), "W/") == etag {
			return true
		}
	}
	return false
}

package page

import (
	"context"
	"strings"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// UpdatePageはトークンを束縛したスペースのページを、JSON Merge Patchで更新する
// (PATCH /api/v1/spaces/{space_identifier}/pages/{page_number})。
// `If-Match` が無ければ428、ページの今のETagと一致しなければ412を返し、
// 更新したページを新しい `ETag` を付けて200で返す
func (h *Handler) UpdatePage(ctx context.Context, request apigen.UpdatePageRequestObject) (apigen.UpdatePageResponseObject, error) {
	if request.Params.IfMatch == nil {
		return nil, apierror.ErrPreconditionRequired
	}
	digests, matchAny := parseIfMatch(*request.Params.IfMatch)

	output, err := h.updateAPIPageUC.Execute(ctx, usecase.UpdateAPIPageInput{
		Principal:       middleware.APIPrincipalFromContext(ctx),
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

	page, err := toAPIPage(output.Page, output.Topic)
	if err != nil {
		return nil, err
	}
	return apigen.UpdatePage200JSONResponse{
		Body:    page,
		Headers: apigen.UpdatePage200ResponseHeaders{ETag: `"` + output.Page.ContentDigest() + `"`},
	}, nil
}

// parseIfMatchは `If-Match` の値 (RFC 9110 §13.1.1) を、照合するページのダイジェスト (ETagの引用符の中身) の並びに変える。
// 値が `*` なら、ページが存在すれば一致とするためmatchAnyにtrueを返す。
// `If-Match` は強い比較で判定するため、弱いエンティティタグ (`W/`) はどのETagとも一致しないものとして除く。
// 引用符で囲まれていないエンティティタグも、どのETagとも一致しないものとして除く
func parseIfMatch(ifMatch string) (digests []string, matchAny bool) {
	if strings.TrimSpace(ifMatch) == "*" {
		return nil, true
	}
	for tag := range strings.SplitSeq(ifMatch, ",") {
		tag = strings.TrimSpace(tag)
		if len(tag) < 2 || !strings.HasPrefix(tag, `"`) || !strings.HasSuffix(tag, `"`) {
			continue
		}
		digests = append(digests, tag[1:len(tag)-1])
	}
	return digests, false
}

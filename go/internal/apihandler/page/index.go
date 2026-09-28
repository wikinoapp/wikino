package page

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/apipagination"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// pageCursorはページの一覧のカーソルの中身。更新日時は複数のページで重なりうるため、主キーと組にして位置を決める
type pageCursor struct {
	ModifiedAt *time.Time    `json:"m"`
	PageID     *model.PageID `json:"i"`
}

// ListPagesはトークンを束縛したスペースのページの一覧を返す (GET /api/v1/spaces/{space_identifier}/pages)
func (h *Handler) ListPages(ctx context.Context, request apigen.ListPagesRequestObject) (apigen.ListPagesResponseObject, error) {
	principal := middleware.APIPrincipalFromContext(ctx)
	// カーソルはハンドラーで検証するため、その前に束縛先を確かめ、異なるスペースでは未存在として扱う。
	// スキーマで表せるパラメーターの違反 (`limit` など) は、ハンドラーより前のリクエスト検証が422を返す。
	// UseCase側でも同じ束縛先を検証し、認可をHandlerだけに委ねない。
	if !principal.IsBoundTo(model.SpaceIdentifier(request.SpaceIdentifier)) {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	after, err := decodePageCursor(ctx, request.Params.Cursor)
	if err != nil {
		return nil, err
	}

	output, err := h.listAPIPagesUC.Execute(ctx, usecase.ListAPIPagesInput{
		Principal:       principal,
		SpaceIdentifier: model.SpaceIdentifier(request.SpaceIdentifier),
		TopicNumber:     request.Params.TopicNumber,
		ModifiedSince:   request.Params.ModifiedSince,
		After:           after,
		Limit:           apipagination.Limit(request.Params.Limit),
	})
	if err != nil {
		return nil, err
	}

	items := make([]apigen.Page, len(output.Pages))
	for i, page := range output.Pages {
		items[i], err = toAPIPage(page, output.Topics[page.TopicID])
		if err != nil {
			return nil, err
		}
	}

	res := apigen.ListPages200JSONResponse{Body: apigen.PageList{Items: items}}
	if output.HasNext {
		last := output.Pages[len(output.Pages)-1]
		modifiedAt := last.ModifiedAt.UTC()
		nextCursor, err := apipagination.EncodeCursor(pageCursor{ModifiedAt: &modifiedAt, PageID: &last.ID})
		if err != nil {
			return nil, err
		}
		res.Body.NextCursor = &nextCursor
		if link := apipagination.NextLink(ctx, nextCursor); link != "" {
			res.Headers.Link = &link
		}
	}
	return res, nil
}

// decodePageCursorはリクエストのカーソルから、前のページの最後のページの位置を取り出す。
// カーソルが無ければnilを返す
func decodePageCursor(ctx context.Context, cursor *string) (*usecase.APIPageCursor, error) {
	if cursor == nil {
		return nil, nil
	}

	invalid := &apierror.ParameterError{Parameter: "cursor", Detail: i18n.T(ctx, "api_error_invalid_cursor")}
	var c pageCursor
	if err := apipagination.DecodeCursor(*cursor, &c); err != nil || c.ModifiedAt == nil || c.PageID == nil {
		return nil, invalid
	}
	// IDはSQLでUUIDとして扱うため、UUIDとして読めないものはここで拒否する
	if _, err := uuid.Parse(string(*c.PageID)); err != nil {
		return nil, invalid
	}
	return &usecase.APIPageCursor{ModifiedAt: *c.ModifiedAt, PageID: *c.PageID}, nil
}

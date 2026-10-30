package draft_page

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

// draftPageCursorは下書きの一覧のカーソルの中身。更新日時は複数の下書きで重なりうるため、主キーと組にして位置を決める
type draftPageCursor struct {
	ModifiedAt  *time.Time         `json:"m"`
	DraftPageID *model.DraftPageID `json:"i"`
}

// ListDraftPagesはトークンの持ち主がトークンの属するスペースに持つ下書きの一覧を返す
// (GET /api/v1/spaces/{space_identifier}/draft_pages)
func (h *Handler) ListDraftPages(ctx context.Context, request apigen.ListDraftPagesRequestObject) (apigen.ListDraftPagesResponseObject, error) {
	principal := middleware.APIPrincipalFromContext(ctx)
	// カーソルはハンドラーで検証するため、その前にトークンが属するスペースかを確かめ、属さないスペースでは未存在として扱う。
	// UseCase側でも同じくトークンが属するスペースかを検証し、認可をHandlerだけに委ねない。
	if !principal.IsBoundTo(model.SpaceIdentifier(request.SpaceIdentifier)) {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	after, err := decodeDraftPageCursor(ctx, request.Params.Cursor)
	if err != nil {
		return nil, err
	}

	output, err := h.listAPIDraftPagesUC.Execute(ctx, usecase.ListAPIDraftPagesInput{
		Principal:       principal,
		SpaceIdentifier: model.SpaceIdentifier(request.SpaceIdentifier),
		After:           after,
		Limit:           apipagination.Limit(request.Params.Limit),
	})
	if err != nil {
		return nil, err
	}

	items := make([]apigen.DraftPageListItem, len(output.DraftPages))
	for i, draft := range output.DraftPages {
		items[i], err = toAPIDraftPageListItem(h.appURL, principal.Space, draft, output.Topics[draft.TopicID])
		if err != nil {
			return nil, err
		}
	}

	res := apigen.ListDraftPages200JSONResponse{Body: apigen.DraftPageList{Items: items}}
	if output.HasNext {
		last := output.DraftPages[len(output.DraftPages)-1]
		modifiedAt := last.ModifiedAt.UTC()
		nextCursor, err := apipagination.EncodeCursor(draftPageCursor{ModifiedAt: &modifiedAt, DraftPageID: &last.ID})
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

// toAPIDraftPageListItemは下書きと所属トピックを、APIの下書きの一覧の項目に変換する。
// 本文の抜粋はここで本文から作る
func toAPIDraftPageListItem(appURL string, space *model.Space, draft *model.DraftPage, topic *model.Topic) (apigen.DraftPageListItem, error) {
	d, err := ToAPIDraftPage(appURL, space, draft, topic)
	if err != nil {
		return apigen.DraftPageListItem{}, err
	}
	return apigen.DraftPageListItem{
		PageNumber:  d.PageNumber,
		TopicNumber: d.TopicNumber,
		TopicName:   d.TopicName,
		Title:       d.Title,
		Excerpt:     model.BodyExcerpt(draft.Body),
		Body:        &d.Body,
		ModifiedAt:  d.ModifiedAt,
		Url:         d.Url,
	}, nil
}

// decodeDraftPageCursorはリクエストのカーソルから、前のページの最後の下書きの位置を取り出す。
// カーソルが無ければnilを返す
func decodeDraftPageCursor(ctx context.Context, cursor *string) (*usecase.APIDraftPageCursor, error) {
	if cursor == nil {
		return nil, nil
	}

	invalid := &apierror.ParameterError{Parameter: "cursor", Detail: i18n.T(ctx, "api_error_invalid_cursor")}
	var c draftPageCursor
	if err := apipagination.DecodeCursor(*cursor, &c); err != nil || c.ModifiedAt == nil || c.DraftPageID == nil {
		return nil, invalid
	}
	// IDはSQLでUUIDとして扱うため、UUIDとして読めないものはここで拒否する
	if _, err := uuid.Parse(string(*c.DraftPageID)); err != nil {
		return nil, invalid
	}
	return &usecase.APIDraftPageCursor{ModifiedAt: *c.ModifiedAt, DraftPageID: *c.DraftPageID}, nil
}

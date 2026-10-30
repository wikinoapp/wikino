package draft_page_revision

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

// draftPageRevisionCursorは下書きの編集履歴のカーソルの中身。作成日時は複数のリビジョンで重なりうるため、主キーと組にして位置を決める
type draftPageRevisionCursor struct {
	CreatedAt  *time.Time                 `json:"c"`
	RevisionID *model.DraftPageRevisionID `json:"i"`
}

// ListDraftPageRevisionsはページについてトークンの持ち主の下書きのリビジョンを新しい順に返す
// (GET /api/v1/spaces/{space_identifier}/pages/{page_number}/draft_page_revisions)
func (h *Handler) ListDraftPageRevisions(ctx context.Context, request apigen.ListDraftPageRevisionsRequestObject) (apigen.ListDraftPageRevisionsResponseObject, error) {
	principal := middleware.APIPrincipalFromContext(ctx)
	// カーソルはハンドラーで検証するため、その前にトークンが属するスペースかを確かめ、属さないスペースでは未存在として扱う。
	// UseCase側でも同じくトークンが属するスペースかを検証し、認可をHandlerだけに委ねない。
	if !principal.IsBoundTo(model.SpaceIdentifier(request.SpaceIdentifier)) {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	after, err := decodeDraftPageRevisionCursor(ctx, request.Params.Cursor)
	if err != nil {
		return nil, err
	}

	output, err := h.listAPIDraftPageRevisionsUC.Execute(ctx, usecase.ListAPIDraftPageRevisionsInput{
		Principal:       principal,
		SpaceIdentifier: model.SpaceIdentifier(request.SpaceIdentifier),
		PageNumber:      request.PageNumber,
		After:           after,
		Limit:           apipagination.Limit(request.Params.Limit),
	})
	if err != nil {
		return nil, err
	}

	items := make([]apigen.DraftPageRevisionListItem, len(output.Revisions))
	for i, revision := range output.Revisions {
		items[i], err = toAPIDraftPageRevisionListItem(revision)
		if err != nil {
			return nil, err
		}
	}

	res := apigen.ListDraftPageRevisions200JSONResponse{Body: apigen.DraftPageRevisionList{Items: items}}
	if output.HasNext {
		last := output.Revisions[len(output.Revisions)-1]
		createdAt := last.CreatedAt.UTC()
		nextCursor, err := apipagination.EncodeCursor(draftPageRevisionCursor{CreatedAt: &createdAt, RevisionID: &last.ID})
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

// toAPIDraftPageRevisionListItemは下書きのリビジョンを、APIの編集履歴の一覧の項目に変換する。
// 本文の抜粋はここで本文から作る
func toAPIDraftPageRevisionListItem(revision *model.DraftPageRevision) (apigen.DraftPageRevisionListItem, error) {
	r, err := toAPIDraftPageRevision(revision)
	if err != nil {
		return apigen.DraftPageRevisionListItem{}, err
	}
	return apigen.DraftPageRevisionListItem{
		Id:        r.Id,
		Title:     r.Title,
		Excerpt:   model.BodyExcerpt(revision.Body),
		Body:      &r.Body,
		CreatedAt: r.CreatedAt,
	}, nil
}

// decodeDraftPageRevisionCursorはリクエストのカーソルから、前のページの最後のリビジョンの位置を取り出す。
// カーソルが無ければnilを返す
func decodeDraftPageRevisionCursor(ctx context.Context, cursor *string) (*usecase.APIDraftPageRevisionCursor, error) {
	if cursor == nil {
		return nil, nil
	}

	invalid := &apierror.ParameterError{Parameter: "cursor", Detail: i18n.T(ctx, "api_error_invalid_cursor")}
	var c draftPageRevisionCursor
	if err := apipagination.DecodeCursor(*cursor, &c); err != nil || c.CreatedAt == nil || c.RevisionID == nil {
		return nil, invalid
	}
	// IDはSQLでUUIDとして扱うため、UUIDとして読めないものはここで拒否する
	if _, err := uuid.Parse(string(*c.RevisionID)); err != nil {
		return nil, invalid
	}
	return &usecase.APIDraftPageRevisionCursor{CreatedAt: *c.CreatedAt, RevisionID: *c.RevisionID}, nil
}

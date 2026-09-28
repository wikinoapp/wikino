package topic

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/apipagination"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// topicCursorはトピックの一覧のカーソルの中身。トピック番号はスペース内で一意なため、番号だけで位置が決まる
type topicCursor struct {
	Number *int32 `json:"n"`
}

// ListTopicsはトークンを束縛したスペースのトピックの一覧を返す (GET /api/v1/spaces/{space_identifier}/topics)
func (h *Handler) ListTopics(ctx context.Context, request apigen.ListTopicsRequestObject) (apigen.ListTopicsResponseObject, error) {
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

	afterNumber, err := decodeTopicCursor(ctx, request.Params.Cursor)
	if err != nil {
		return nil, err
	}

	output, err := h.listAPITopicsUC.Execute(ctx, usecase.ListAPITopicsInput{
		Principal:       principal,
		SpaceIdentifier: model.SpaceIdentifier(request.SpaceIdentifier),
		AfterNumber:     afterNumber,
		Limit:           apipagination.Limit(request.Params.Limit),
	})
	if err != nil {
		return nil, err
	}

	items := make([]apigen.Topic, len(output.Topics))
	for i, topic := range output.Topics {
		items[i], err = toAPITopic(topic)
		if err != nil {
			return nil, err
		}
	}

	res := apigen.ListTopics200JSONResponse{Body: apigen.TopicList{Items: items}}
	if output.HasNext {
		nextCursor, err := apipagination.EncodeCursor(topicCursor{Number: &output.Topics[len(output.Topics)-1].Number})
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

// decodeTopicCursorはリクエストのカーソルから、前のページの最後のトピックの番号を取り出す。
// カーソルが無ければnilを返す
func decodeTopicCursor(ctx context.Context, cursor *string) (*int32, error) {
	if cursor == nil {
		return nil, nil
	}

	var c topicCursor
	if err := apipagination.DecodeCursor(*cursor, &c); err != nil || c.Number == nil {
		return nil, &apierror.ParameterError{Parameter: "cursor", Detail: i18n.T(ctx, "api_error_invalid_cursor")}
	}
	return c.Number, nil
}

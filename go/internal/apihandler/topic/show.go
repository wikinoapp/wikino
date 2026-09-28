package topic

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// GetTopicはトークンを束縛したスペースのトピックを返す (GET /api/v1/spaces/{space_identifier}/topics/{topic_number})
func (h *Handler) GetTopic(ctx context.Context, request apigen.GetTopicRequestObject) (apigen.GetTopicResponseObject, error) {
	output, err := h.getAPITopicUC.Execute(ctx, usecase.GetAPITopicInput{
		Principal:       middleware.APIPrincipalFromContext(ctx),
		SpaceIdentifier: model.SpaceIdentifier(request.SpaceIdentifier),
		TopicNumber:     request.TopicNumber,
	})
	if err != nil {
		return nil, err
	}

	topic, err := toAPITopic(output.Topic)
	if err != nil {
		return nil, err
	}
	return apigen.GetTopic200JSONResponse{Body: topic}, nil
}

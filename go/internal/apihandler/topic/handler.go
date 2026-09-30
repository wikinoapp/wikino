// Package topicは公開Web APIのトピックのハンドラーを提供する
package topic

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはトピックに関するAPIを担当するハンドラー
type Handler struct {
	listAPITopicsUC *usecase.ListAPITopicsUsecase
	getAPITopicUC   *usecase.GetAPITopicUsecase
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(listAPITopicsUC *usecase.ListAPITopicsUsecase, getAPITopicUC *usecase.GetAPITopicUsecase) *Handler {
	return &Handler{
		listAPITopicsUC: listAPITopicsUC,
		getAPITopicUC:   getAPITopicUC,
	}
}

// toAPITopicはトピックをAPIの応答の形に変換する
func toAPITopic(topic *model.Topic) (apigen.Topic, error) {
	id, err := uuid.Parse(string(topic.ID))
	if err != nil {
		return apigen.Topic{}, fmt.Errorf("トピックIDをUUIDとして解釈できません: %w", err)
	}

	return apigen.Topic{
		Id:          id,
		Number:      topic.Number,
		Name:        topic.Name,
		Description: topic.Description,
		Visibility:  apigen.TopicVisibility(topic.Visibility.String()),
	}, nil
}

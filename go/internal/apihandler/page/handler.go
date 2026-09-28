// Package pageは公開Web APIのページのハンドラーを提供する
package page

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはページに関するAPIを担当するハンドラー
type Handler struct {
	listAPIPagesUC  *usecase.ListAPIPagesUsecase
	getAPIPageUC    *usecase.GetAPIPageUsecase
	createAPIPageUC *usecase.CreateAPIPageUsecase
	updateAPIPageUC *usecase.UpdateAPIPageUsecase
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(
	listAPIPagesUC *usecase.ListAPIPagesUsecase,
	getAPIPageUC *usecase.GetAPIPageUsecase,
	createAPIPageUC *usecase.CreateAPIPageUsecase,
	updateAPIPageUC *usecase.UpdateAPIPageUsecase,
) *Handler {
	return &Handler{
		listAPIPagesUC:  listAPIPagesUC,
		getAPIPageUC:    getAPIPageUC,
		createAPIPageUC: createAPIPageUC,
		updateAPIPageUC: updateAPIPageUC,
	}
}

// toAPIPageはページと所属トピックをAPIの応答の形に変換する
func toAPIPage(page *model.Page, topic *model.Topic) (apigen.Page, error) {
	id, err := uuid.Parse(string(page.ID))
	if err != nil {
		return apigen.Page{}, fmt.Errorf("ページIDをUUIDとして解釈できません: %w", err)
	}
	if topic == nil {
		return apigen.Page{}, fmt.Errorf("ページ %s の所属トピックがありません", page.ID)
	}

	return apigen.Page{
		Id:          id,
		Number:      int32(page.Number),
		TopicNumber: topic.Number,
		Title:       page.Title,
		Body:        page.Body,
		ModifiedAt:  page.ModifiedAt.UTC(),
	}, nil
}

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
	// appURLはWikinoのオリジン (例: `https://example.com`)。応答の `url` を組み立てるのに使う
	appURL          string
	listAPIPagesUC  *usecase.ListAPIPagesUsecase
	getAPIPageUC    *usecase.GetAPIPageUsecase
	createAPIPageUC *usecase.CreateAPIPageUsecase
	updateAPIPageUC *usecase.UpdateAPIPageUsecase
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(
	appURL string,
	listAPIPagesUC *usecase.ListAPIPagesUsecase,
	getAPIPageUC *usecase.GetAPIPageUsecase,
	createAPIPageUC *usecase.CreateAPIPageUsecase,
	updateAPIPageUC *usecase.UpdateAPIPageUsecase,
) *Handler {
	return &Handler{
		appURL:          appURL,
		listAPIPagesUC:  listAPIPagesUC,
		getAPIPageUC:    getAPIPageUC,
		createAPIPageUC: createAPIPageUC,
		updateAPIPageUC: updateAPIPageUC,
	}
}

// ToAPIPageはページと所属トピックをAPIの応答の形に変換する。
// appURLはWikinoのオリジン、spaceはページが属するスペースで、Web画面で開くURL (`url`) の組み立てに使う
func ToAPIPage(appURL string, space *model.Space, page *model.Page, topic *model.Topic) (apigen.Page, error) {
	id, err := uuid.Parse(string(page.ID))
	if err != nil {
		return apigen.Page{}, fmt.Errorf("ページIDをUUIDとして解釈できません: %w", err)
	}
	if topic == nil {
		return apigen.Page{}, fmt.Errorf("ページ %s の所属トピックがありません", page.ID)
	}
	if space == nil || space.ID != page.SpaceID {
		return apigen.Page{}, fmt.Errorf("ページ %s の所属スペースがありません", page.ID)
	}

	return apigen.Page{
		Id:          id,
		Number:      int32(page.Number),
		TopicNumber: topic.Number,
		TopicName:   topic.Name,
		Title:       page.Title,
		Body:        page.Body,
		Status:      toAPIPageStatus(page),
		ModifiedAt:  page.ModifiedAt.UTC(),
		Url:         model.PageWebURL(appURL, space, page.Number),
	}, nil
}

// toAPIPageStatusはページの公開状態をAPIの値に変換する
func toAPIPageStatus(page *model.Page) apigen.PageStatus {
	if page.IsPublished() {
		return apigen.PageStatusPublished
	}
	return apigen.PageStatusUnpublished
}

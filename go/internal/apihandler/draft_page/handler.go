// Package draft_pageは公開Web APIの下書きのハンドラーを提供する
package draft_page

import (
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerは下書きに関するAPIを担当するハンドラー
type Handler struct {
	// appURLはWikinoのオリジン (例: `https://example.com`)。応答の `url` を組み立てるのに使う
	appURL               string
	listAPIDraftPagesUC  *usecase.ListAPIDraftPagesUsecase
	getAPIDraftPageUC    *usecase.GetAPIDraftPageUsecase
	createAPIDraftPageUC *usecase.CreateAPIDraftPageUsecase
	updateAPIDraftPageUC *usecase.UpdateAPIDraftPageUsecase
	deleteAPIDraftPageUC *usecase.DeleteAPIDraftPageUsecase
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(
	appURL string,
	listAPIDraftPagesUC *usecase.ListAPIDraftPagesUsecase,
	getAPIDraftPageUC *usecase.GetAPIDraftPageUsecase,
	createAPIDraftPageUC *usecase.CreateAPIDraftPageUsecase,
	updateAPIDraftPageUC *usecase.UpdateAPIDraftPageUsecase,
	deleteAPIDraftPageUC *usecase.DeleteAPIDraftPageUsecase,
) *Handler {
	return &Handler{
		appURL:               appURL,
		listAPIDraftPagesUC:  listAPIDraftPagesUC,
		getAPIDraftPageUC:    getAPIDraftPageUC,
		createAPIDraftPageUC: createAPIDraftPageUC,
		updateAPIDraftPageUC: updateAPIDraftPageUC,
		deleteAPIDraftPageUC: deleteAPIDraftPageUC,
	}
}

// ToAPIDraftPageは下書きと所属トピックをAPIの応答の形に変換する。
// appURLはWikinoのオリジン、spaceは下書きが属するスペースで、Web画面の編集画面のURL (`url`) の組み立てに使う
func ToAPIDraftPage(appURL string, space *model.Space, draft *model.DraftPage, topic *model.Topic) (apigen.DraftPage, error) {
	if draft.Page == nil {
		return apigen.DraftPage{}, fmt.Errorf("下書き %s のページがありません", draft.ID)
	}
	if topic == nil {
		return apigen.DraftPage{}, fmt.Errorf("下書き %s の所属トピックがありません", draft.ID)
	}
	if space == nil || space.ID != draft.SpaceID {
		return apigen.DraftPage{}, fmt.Errorf("下書き %s の所属スペースがありません", draft.ID)
	}

	return apigen.DraftPage{
		PageNumber:  int32(draft.Page.Number),
		TopicNumber: topic.Number,
		TopicName:   topic.Name,
		Title:       draft.Title,
		Body:        draft.Body,
		ModifiedAt:  draft.ModifiedAt.UTC(),
		Url:         model.PageEditWebURL(appURL, space, draft.Page.Number),
	}, nil
}

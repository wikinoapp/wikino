// Package draft_page_revisionは公開Web APIの下書きの編集履歴のハンドラーを提供する
package draft_page_revision

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerは下書きの編集履歴に関するAPIを担当するハンドラー
type Handler struct {
	listAPIDraftPageRevisionsUC *usecase.ListAPIDraftPageRevisionsUsecase
	getAPIDraftPageRevisionUC   *usecase.GetAPIDraftPageRevisionUsecase
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(
	listAPIDraftPageRevisionsUC *usecase.ListAPIDraftPageRevisionsUsecase,
	getAPIDraftPageRevisionUC *usecase.GetAPIDraftPageRevisionUsecase,
) *Handler {
	return &Handler{
		listAPIDraftPageRevisionsUC: listAPIDraftPageRevisionsUC,
		getAPIDraftPageRevisionUC:   getAPIDraftPageRevisionUC,
	}
}

// toAPIDraftPageRevisionは下書きのリビジョンをAPIの応答の形に変換する。
// リビジョンはタイトルの無い下書きを空のタイトルで記録しているため、下書きの応答と揃えて `null` で返す
func toAPIDraftPageRevision(revision *model.DraftPageRevision) (apigen.DraftPageRevision, error) {
	id, err := uuid.Parse(string(revision.ID))
	if err != nil {
		return apigen.DraftPageRevision{}, fmt.Errorf("下書きのリビジョンのID %q をUUIDとして読めません: %w", revision.ID, err)
	}

	var title *string
	if revision.Title != "" {
		title = &revision.Title
	}
	return apigen.DraftPageRevision{
		Id:        id,
		Title:     title,
		Body:      revision.Body,
		CreatedAt: revision.CreatedAt.UTC(),
	}, nil
}

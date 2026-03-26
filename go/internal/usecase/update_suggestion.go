package usecase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// UpdateSuggestionUsecase は編集提案更新ユースケース
type UpdateSuggestionUsecase struct {
	db             *sql.DB
	suggestionRepo *repository.SuggestionRepository
	topicRepo      *repository.TopicRepository
	pageRepo       *repository.PageRepository
}

// NewUpdateSuggestionUsecase は UpdateSuggestionUsecase を生成する
func NewUpdateSuggestionUsecase(
	db *sql.DB,
	suggestionRepo *repository.SuggestionRepository,
	topicRepo *repository.TopicRepository,
	pageRepo *repository.PageRepository,
) *UpdateSuggestionUsecase {
	return &UpdateSuggestionUsecase{
		db:             db,
		suggestionRepo: suggestionRepo,
		topicRepo:      topicRepo,
		pageRepo:       pageRepo,
	}
}

// UpdateSuggestionInput は編集提案更新の入力パラメータ
type UpdateSuggestionInput struct {
	SuggestionID     model.SuggestionID
	SpaceID          model.SpaceID
	SpaceIdentifier  model.SpaceIdentifier
	CurrentTopicName string
	Title            string
	Body             string
}

// UpdateSuggestionOutput は編集提案更新の出力パラメータ
type UpdateSuggestionOutput struct {
	Suggestion *model.Suggestion
}

// Execute は編集提案を更新する
func (uc *UpdateSuggestionUsecase) Execute(ctx context.Context, input UpdateSuggestionInput) (*UpdateSuggestionOutput, error) {
	// トランザクション前: データ取得・計算
	bodyHTML, err := uc.renderBodyHTML(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("本文HTMLの生成に失敗しました: %w", err)
	}

	// トランザクション: 永続化のみ
	return uc.updateSuggestion(ctx, input, bodyHTML)
}

// updateSuggestion はトランザクション内で編集提案を更新する
func (uc *UpdateSuggestionUsecase) updateSuggestion(ctx context.Context, input UpdateSuggestionInput, bodyHTML string) (*UpdateSuggestionOutput, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	suggestionRepo := uc.suggestionRepo.WithTx(tx)

	suggestion, err := suggestionRepo.Update(ctx, repository.UpdateSuggestionInput{
		ID:       input.SuggestionID,
		SpaceID:  input.SpaceID,
		Title:    input.Title,
		Body:     input.Body,
		BodyHTML: bodyHTML,
	})
	if err != nil {
		return nil, fmt.Errorf("編集提案の更新に失敗しました: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &UpdateSuggestionOutput{Suggestion: suggestion}, nil
}

// renderBodyHTML は本文のMarkdownをHTMLに変換し、Wikiリンクを解決する
func (uc *UpdateSuggestionUsecase) renderBodyHTML(ctx context.Context, input UpdateSuggestionInput) (string, error) {
	bodyHTML := markup.RenderMarkdown(input.Body)

	pageLocations, err := resolveLinkedPages(ctx, input.Body, input.CurrentTopicName, input.SpaceID, uc.topicRepo, uc.pageRepo)
	if err != nil {
		return "", fmt.Errorf("wikiリンクの解析に失敗しました: %w", err)
	}
	if len(pageLocations) > 0 {
		bodyHTML = markup.ReplaceWikilinks(bodyHTML, input.CurrentTopicName, input.SpaceIdentifier, pageLocations)
	}

	bodyHTML = markup.WrapStandaloneImageLinks(bodyHTML)

	return bodyHTML, nil
}

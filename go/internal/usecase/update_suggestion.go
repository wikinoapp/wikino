package usecase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// UpdateSuggestionUsecase は編集提案更新ユースケース
type UpdateSuggestionUsecase struct {
	db              *sql.DB
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	suggestionRepo  *repository.SuggestionRepository
	pageRepo        *repository.PageRepository
	updateValidator *validator.SuggestionUpdateValidator
}

// NewUpdateSuggestionUsecase は UpdateSuggestionUsecase を生成する
func NewUpdateSuggestionUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	suggestionRepo *repository.SuggestionRepository,
	pageRepo *repository.PageRepository,
	updateValidator *validator.SuggestionUpdateValidator,
) *UpdateSuggestionUsecase {
	return &UpdateSuggestionUsecase{
		db:              db,
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
		suggestionRepo:  suggestionRepo,
		pageRepo:        pageRepo,
		updateValidator: updateValidator,
	}
}

// UpdateSuggestionInput は編集提案更新の入力パラメータ
type UpdateSuggestionInput struct {
	SpaceIdentifier  model.SpaceIdentifier
	SuggestionNumber model.SuggestionNumber
	UserID           model.UserID
	Title            string
	Body             string
}

// UpdateSuggestionOutput は編集提案更新の出力パラメータ
type UpdateSuggestionOutput struct {
	Suggestion *model.Suggestion
}

// Execute は編集提案を更新する
func (uc *UpdateSuggestionUsecase) Execute(ctx context.Context, input UpdateSuggestionInput) (*UpdateSuggestionOutput, error) {
	// 1. データ取得
	space, spaceMember, topic, suggestion, err := uc.fetchData(ctx, input)
	if err != nil {
		return nil, err
	}

	// 2. 認可チェック
	if err := uc.authorize(ctx, space, spaceMember, topic, suggestion); err != nil {
		return nil, err
	}

	// 3. バリデーション
	if err := uc.updateValidator.Validate(ctx, validator.SuggestionUpdateValidatorInput{
		Title: input.Title,
		Body:  input.Body,
	}); err != nil {
		return nil, err
	}

	// 4. ビジネスロジック（トランザクション前）
	bodyHTML, err := uc.renderBodyHTML(ctx, input.Body, topic.Name, space.ID, input.SpaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("本文HTMLの生成に失敗しました: %w", err)
	}

	// 5. 永続化（トランザクション）
	return uc.updateSuggestion(ctx, suggestion.ID, space.ID, input.Title, input.Body, bodyHTML)
}

func (uc *UpdateSuggestionUsecase) fetchData(ctx context.Context, input UpdateSuggestionInput) (*model.Space, *model.SpaceMember, *model.Topic, *model.Suggestion, error) {
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, nil, nil, nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	suggestion, err := uc.suggestionRepo.FindBySpaceAndNumber(ctx, space.ID, input.SuggestionNumber)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("編集提案の取得に失敗: %w", err)
	}
	if suggestion == nil {
		return nil, nil, nil, nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	topic, err := uc.topicRepo.FindBySpaceAndID(ctx, space.ID, suggestion.TopicID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("トピックの取得に失敗: %w", err)
	}
	if topic == nil {
		return nil, nil, nil, nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	spaceMember, err := uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, input.UserID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}

	return space, spaceMember, topic, suggestion, nil
}

func (uc *UpdateSuggestionUsecase) authorize(ctx context.Context, space *model.Space, spaceMember *model.SpaceMember, topic *model.Topic, suggestion *model.Suggestion) error {
	if spaceMember == nil {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	var topicMember *model.TopicMember
	if spaceMember.Role != model.SpaceMemberRoleOwner {
		var err error
		topicMember, err = uc.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, topic.ID)
		if err != nil {
			return fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
		}
	}

	topicPolicy := policy.NewTopicPolicy(spaceMember, topicMember)
	if !topicPolicy.CanUpdateSuggestion(suggestion) {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	return nil
}

// updateSuggestion はトランザクション内で編集提案を更新する
func (uc *UpdateSuggestionUsecase) updateSuggestion(ctx context.Context, suggestionID model.SuggestionID, spaceID model.SpaceID, title, body, bodyHTML string) (*UpdateSuggestionOutput, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	suggestionRepo := uc.suggestionRepo.WithTx(tx)

	suggestion, err := suggestionRepo.Update(ctx, repository.UpdateSuggestionInput{
		ID:       suggestionID,
		SpaceID:  spaceID,
		Title:    title,
		Body:     body,
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
func (uc *UpdateSuggestionUsecase) renderBodyHTML(ctx context.Context, body, currentTopicName string, spaceID model.SpaceID, spaceIdentifier model.SpaceIdentifier) (string, error) {
	bodyHTML := markup.RenderMarkdown(body)

	pageLocations, err := resolveLinkedPages(ctx, body, currentTopicName, spaceID, uc.topicRepo, uc.pageRepo)
	if err != nil {
		return "", fmt.Errorf("wikiリンクの解析に失敗しました: %w", err)
	}
	if len(pageLocations) > 0 {
		bodyHTML = markup.ReplaceWikilinks(bodyHTML, currentTopicName, spaceIdentifier, pageLocations)
	}

	bodyHTML = markup.WrapStandaloneImageLinks(bodyHTML)

	return bodyHTML, nil
}

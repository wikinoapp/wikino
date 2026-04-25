package usecase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// AddSuggestionPageUsecase は編集提案ページ追加ユースケース
type AddSuggestionPageUsecase struct {
	db                         *sql.DB
	spaceRepo                  *repository.SpaceRepository
	spaceMemberRepo            *repository.SpaceMemberRepository
	topicMemberRepo            *repository.TopicMemberRepository
	suggestionRepo             *repository.SuggestionRepository
	suggestionPageRepo         *repository.SuggestionPageRepository
	suggestionPageRevisionRepo *repository.SuggestionPageRevisionRepository
	draftPageRepo              *repository.DraftPageRepository
	pageRevisionRepo           *repository.PageRevisionRepository
	createValidator            *validator.SuggestionPageCreateValidator
}

// NewAddSuggestionPageUsecase は AddSuggestionPageUsecase を生成する
func NewAddSuggestionPageUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	suggestionRepo *repository.SuggestionRepository,
	suggestionPageRepo *repository.SuggestionPageRepository,
	suggestionPageRevisionRepo *repository.SuggestionPageRevisionRepository,
	draftPageRepo *repository.DraftPageRepository,
	pageRevisionRepo *repository.PageRevisionRepository,
	createValidator *validator.SuggestionPageCreateValidator,
) *AddSuggestionPageUsecase {
	return &AddSuggestionPageUsecase{
		db:                         db,
		spaceRepo:                  spaceRepo,
		spaceMemberRepo:            spaceMemberRepo,
		topicMemberRepo:            topicMemberRepo,
		suggestionRepo:             suggestionRepo,
		suggestionPageRepo:         suggestionPageRepo,
		suggestionPageRevisionRepo: suggestionPageRevisionRepo,
		draftPageRepo:              draftPageRepo,
		pageRevisionRepo:           pageRevisionRepo,
		createValidator:            createValidator,
	}
}

// AddSuggestionPageInput は編集提案ページ追加の入力パラメータ
type AddSuggestionPageInput struct {
	SpaceIdentifier  model.SpaceIdentifier
	SuggestionNumber model.SuggestionNumber
	UserID           model.UserID
	DraftPageIDs     []model.DraftPageID
}

// AddSuggestionPageOutput は編集提案ページ追加の出力パラメータ
type AddSuggestionPageOutput struct {
	Suggestion *model.Suggestion
}

// Execute は編集提案にページを追加する
func (uc *AddSuggestionPageUsecase) Execute(ctx context.Context, input AddSuggestionPageInput) (*AddSuggestionPageOutput, error) {
	// 1. データ取得
	space, spaceMember, suggestion, err := uc.fetchData(ctx, input)
	if err != nil {
		return nil, err
	}

	// 2. 認可チェック
	if err := uc.authorize(ctx, space, spaceMember, suggestion); err != nil {
		return nil, err
	}

	// 3. バリデーション
	draftPages, err := uc.createValidator.Validate(ctx, validator.SuggestionPageCreateValidatorInput{
		DraftPageIDs:  input.DraftPageIDs,
		SpaceMemberID: spaceMember.ID,
		TopicID:       suggestion.TopicID,
		SpaceID:       space.ID,
		SuggestionID:  suggestion.ID,
	})
	if err != nil {
		return nil, err
	}

	// 4. ビジネスロジック（トランザクション前）
	pageRevisions, err := fetchLatestPageRevisions(ctx, draftPages, space.ID, uc.pageRevisionRepo)
	if err != nil {
		return nil, err
	}

	// 5. 永続化（トランザクション）
	return uc.addSuggestionPages(ctx, space.ID, spaceMember.ID, suggestion, draftPages, pageRevisions)
}

func (uc *AddSuggestionPageUsecase) fetchData(ctx context.Context, input AddSuggestionPageInput) (*model.Space, *model.SpaceMember, *model.Suggestion, error) {
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, nil, nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	suggestion, err := uc.suggestionRepo.FindBySpaceAndNumber(ctx, space.ID, input.SuggestionNumber)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("編集提案の取得に失敗: %w", err)
	}
	if suggestion == nil {
		return nil, nil, nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	spaceMember, err := uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, input.UserID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}

	return space, spaceMember, suggestion, nil
}

func (uc *AddSuggestionPageUsecase) authorize(ctx context.Context, space *model.Space, spaceMember *model.SpaceMember, suggestion *model.Suggestion) error {
	if spaceMember == nil {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	var topicMember *model.TopicMember
	if !model.HasScope(spaceMember.Scopes, model.ScopeSpaceAdmin) {
		var err error
		topicMember, err = uc.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, suggestion.TopicID)
		if err != nil {
			return fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
		}
	}

	authorizer := newAuthorizer(spaceMember, topicMember)
	if !authorizer.CanAddSuggestionPage(suggestion) {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	return nil
}

func (uc *AddSuggestionPageUsecase) addSuggestionPages(
	ctx context.Context,
	spaceID model.SpaceID,
	spaceMemberID model.SpaceMemberID,
	suggestion *model.Suggestion,
	draftPages []*model.DraftPage,
	pageRevisions map[model.PageID]*model.PageRevision,
) (*AddSuggestionPageOutput, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	suggestionPageRepo := uc.suggestionPageRepo.WithTx(tx)
	suggestionPageRevisionRepo := uc.suggestionPageRevisionRepo.WithTx(tx)
	draftPageRepo := uc.draftPageRepo.WithTx(tx)

	for _, draftPage := range draftPages {
		var pageRevisionID *model.PageRevisionID
		if latestRevision := pageRevisions[draftPage.PageID]; latestRevision != nil {
			pageRevisionID = &latestRevision.ID
		}

		_, err = createSuggestionPageFromDraftPage(ctx, createSuggestionPageInput{
			SpaceID:        spaceID,
			SuggestionID:   suggestion.ID,
			SpaceMemberID:  spaceMemberID,
			DraftPage:      draftPage,
			PageRevisionID: pageRevisionID,
		}, suggestionPageRepo, suggestionPageRevisionRepo, draftPageRepo)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &AddSuggestionPageOutput{
		Suggestion: suggestion,
	}, nil
}

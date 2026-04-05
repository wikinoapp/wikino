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

// CreateSuggestionUsecase は編集提案作成ユースケース
type CreateSuggestionUsecase struct {
	db                         *sql.DB
	spaceRepo                  *repository.SpaceRepository
	spaceMemberRepo            *repository.SpaceMemberRepository
	topicRepo                  *repository.TopicRepository
	topicMemberRepo            *repository.TopicMemberRepository
	suggestionRepo             *repository.SuggestionRepository
	suggestionPageRepo         *repository.SuggestionPageRepository
	suggestionPageRevisionRepo *repository.SuggestionPageRevisionRepository
	draftPageRepo              *repository.DraftPageRepository
	pageRepo                   *repository.PageRepository
	pageRevisionRepo           *repository.PageRevisionRepository
	createValidator            *validator.SuggestionCreateValidator
}

// NewCreateSuggestionUsecase は CreateSuggestionUsecase を生成する
func NewCreateSuggestionUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	suggestionRepo *repository.SuggestionRepository,
	suggestionPageRepo *repository.SuggestionPageRepository,
	suggestionPageRevisionRepo *repository.SuggestionPageRevisionRepository,
	draftPageRepo *repository.DraftPageRepository,
	pageRepo *repository.PageRepository,
	pageRevisionRepo *repository.PageRevisionRepository,
	createValidator *validator.SuggestionCreateValidator,
) *CreateSuggestionUsecase {
	return &CreateSuggestionUsecase{
		db:                         db,
		spaceRepo:                  spaceRepo,
		spaceMemberRepo:            spaceMemberRepo,
		topicRepo:                  topicRepo,
		topicMemberRepo:            topicMemberRepo,
		suggestionRepo:             suggestionRepo,
		suggestionPageRepo:         suggestionPageRepo,
		suggestionPageRevisionRepo: suggestionPageRevisionRepo,
		draftPageRepo:              draftPageRepo,
		pageRepo:                   pageRepo,
		pageRevisionRepo:           pageRevisionRepo,
		createValidator:            createValidator,
	}
}

// CreateSuggestionInput は編集提案作成の入力パラメータ
type CreateSuggestionInput struct {
	SpaceIdentifier model.SpaceIdentifier
	TopicNumber     int32
	UserID          model.UserID
	Title           string
	Body            string
	DraftPageIDs    []model.DraftPageID
}

// CreateSuggestionOutput は編集提案作成の出力パラメータ
type CreateSuggestionOutput struct {
	Suggestion *model.Suggestion
}

// Execute は編集提案を作成する
func (uc *CreateSuggestionUsecase) Execute(ctx context.Context, input CreateSuggestionInput) (*CreateSuggestionOutput, error) {
	// 1. データ取得
	space, spaceMember, topic, err := uc.fetchData(ctx, input)
	if err != nil {
		return nil, err
	}

	// 2. 認可チェック
	if err := uc.authorize(ctx, space, spaceMember, topic); err != nil {
		return nil, err
	}

	// 3. バリデーション
	draftPages, err := uc.createValidator.Validate(ctx, validator.SuggestionCreateValidatorInput{
		Title:         input.Title,
		Body:          input.Body,
		DraftPageIDs:  input.DraftPageIDs,
		SpaceMemberID: spaceMember.ID,
		TopicID:       topic.ID,
		SpaceID:       space.ID,
	})
	if err != nil {
		return nil, err
	}

	// 4. ビジネスロジック（トランザクション前）
	bodyHTML, err := uc.renderBodyHTML(ctx, input.Body, topic.Name, space.ID, input.SpaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("本文HTMLの生成に失敗しました: %w", err)
	}

	pageRevisions, err := fetchLatestPageRevisions(ctx, draftPages, space.ID, uc.pageRevisionRepo)
	if err != nil {
		return nil, err
	}

	// 5. 永続化（トランザクション）
	return uc.createSuggestion(ctx, createSuggestionInput{
		SpaceID:       space.ID,
		TopicID:       topic.ID,
		SpaceMemberID: spaceMember.ID,
		Title:         input.Title,
		Body:          input.Body,
		BodyHTML:      bodyHTML,
		DraftPages:    draftPages,
		PageRevisions: pageRevisions,
	})
}

func (uc *CreateSuggestionUsecase) fetchData(ctx context.Context, input CreateSuggestionInput) (*model.Space, *model.SpaceMember, *model.Topic, error) {
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

	topic, err := uc.topicRepo.FindBySpaceAndNumber(ctx, space.ID, input.TopicNumber)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("トピックの取得に失敗: %w", err)
	}
	if topic == nil {
		return nil, nil, nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	spaceMember, err := uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, input.UserID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}

	return space, spaceMember, topic, nil
}

func (uc *CreateSuggestionUsecase) authorize(ctx context.Context, space *model.Space, spaceMember *model.SpaceMember, topic *model.Topic) error {
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
	if !topicPolicy.CanCreateSuggestion(topic) {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	return nil
}

// renderBodyHTML は本文のMarkdownをHTMLに変換し、Wikiリンクを解決する
func (uc *CreateSuggestionUsecase) renderBodyHTML(ctx context.Context, body, currentTopicName string, spaceID model.SpaceID, spaceIdentifier model.SpaceIdentifier) (string, error) {
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

// createSuggestionInput はトランザクション内で編集提案を作成するための入力パラメータ
type createSuggestionInput struct {
	SpaceID       model.SpaceID
	TopicID       model.TopicID
	SpaceMemberID model.SpaceMemberID
	Title         string
	Body          string
	BodyHTML      string
	DraftPages    []*model.DraftPage
	PageRevisions map[model.PageID]*model.PageRevision
}

// createSuggestion はトランザクション内で編集提案を作成する
func (uc *CreateSuggestionUsecase) createSuggestion(ctx context.Context, input createSuggestionInput) (*CreateSuggestionOutput, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	suggestionRepo := uc.suggestionRepo.WithTx(tx)
	suggestionPageRepo := uc.suggestionPageRepo.WithTx(tx)
	suggestionPageRevisionRepo := uc.suggestionPageRevisionRepo.WithTx(tx)
	draftPageRepo := uc.draftPageRepo.WithTx(tx)

	// 1. スペース内の次の編集提案番号を取得
	nextNumber, err := suggestionRepo.GetNextNumber(ctx, input.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("次の編集提案番号の取得に失敗しました: %w", err)
	}

	// 2. 編集提案を作成
	suggestion, err := suggestionRepo.Create(ctx, repository.CreateSuggestionInput{
		SpaceID:              input.SpaceID,
		TopicID:              input.TopicID,
		CreatedSpaceMemberID: input.SpaceMemberID,
		Number:               nextNumber,
		Title:                input.Title,
		Body:                 input.Body,
		BodyHTML:             input.BodyHTML,
		Status:               model.SuggestionStatusOpen,
	})
	if err != nil {
		return nil, fmt.Errorf("編集提案の作成に失敗しました: %w", err)
	}

	// 3. 各下書きページからSuggestionPageとSuggestionPageRevisionを作成
	for _, draftPage := range input.DraftPages {
		var pageRevisionID *model.PageRevisionID
		if latestRevision := input.PageRevisions[draftPage.PageID]; latestRevision != nil {
			pageRevisionID = &latestRevision.ID
		}

		_, err = createSuggestionPageFromDraftPage(ctx, createSuggestionPageInput{
			SpaceID:        input.SpaceID,
			SuggestionID:   suggestion.ID,
			SpaceMemberID:  input.SpaceMemberID,
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

	return &CreateSuggestionOutput{
		Suggestion: suggestion,
	}, nil
}

// resolveLinkedPages はWikiリンクを解析し、既存ページへのリンク情報を返す。
// resolveAndCreateLinkedPagesと異なり、リンク先ページの自動作成は行わない。
func resolveLinkedPages(
	ctx context.Context,
	body string,
	currentTopicName string,
	spaceID model.SpaceID,
	topicRepo *repository.TopicRepository,
	pageRepo *repository.PageRepository,
) ([]markup.PageLocation, error) {
	keys := markup.ScanWikilinks(body, currentTopicName)
	if len(keys) == 0 {
		return nil, nil
	}

	topicNames := uniqueTopicNames(keys)
	topics, err := topicRepo.FindBySpaceAndNames(ctx, spaceID, topicNames)
	if err != nil {
		return nil, err
	}
	topicMap := make(map[string]*model.Topic, len(topics))
	for _, t := range topics {
		topicMap[t.Name] = t
	}

	var pageLocations []markup.PageLocation
	seen := make(map[string]bool)

	for _, key := range keys {
		lookupKey := key.TopicName + "/" + key.PageTitle
		if seen[lookupKey] {
			continue
		}
		seen[lookupKey] = true

		topic := topicMap[key.TopicName]
		if topic == nil {
			continue
		}

		page, err := pageRepo.FindByTopicAndTitle(ctx, topic.ID, key.PageTitle, spaceID)
		if err != nil {
			return nil, err
		}
		if page == nil {
			continue
		}

		pageTitle := key.PageTitle
		if page.Title != nil {
			pageTitle = *page.Title
		}
		pageLocations = append(pageLocations, markup.PageLocation{
			Key:        key,
			TopicName:  topic.Name,
			PageID:     page.ID,
			PageNumber: int(page.Number),
			PageTitle:  pageTitle,
		})
	}

	return pageLocations, nil
}

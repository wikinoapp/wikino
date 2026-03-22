package usecase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// CreateSuggestionUsecase は編集提案作成ユースケース
type CreateSuggestionUsecase struct {
	db                         *sql.DB
	suggestionRepo             *repository.SuggestionRepository
	suggestionPageRepo         *repository.SuggestionPageRepository
	suggestionPageRevisionRepo *repository.SuggestionPageRevisionRepository
	draftPageRepo              *repository.DraftPageRepository
	topicRepo                  *repository.TopicRepository
	pageRepo                   *repository.PageRepository
	pageRevisionRepo           *repository.PageRevisionRepository
}

// NewCreateSuggestionUsecase は CreateSuggestionUsecase を生成する
func NewCreateSuggestionUsecase(
	db *sql.DB,
	suggestionRepo *repository.SuggestionRepository,
	suggestionPageRepo *repository.SuggestionPageRepository,
	suggestionPageRevisionRepo *repository.SuggestionPageRevisionRepository,
	draftPageRepo *repository.DraftPageRepository,
	topicRepo *repository.TopicRepository,
	pageRepo *repository.PageRepository,
	pageRevisionRepo *repository.PageRevisionRepository,
) *CreateSuggestionUsecase {
	return &CreateSuggestionUsecase{
		db:                         db,
		suggestionRepo:             suggestionRepo,
		suggestionPageRepo:         suggestionPageRepo,
		suggestionPageRevisionRepo: suggestionPageRevisionRepo,
		draftPageRepo:              draftPageRepo,
		topicRepo:                  topicRepo,
		pageRepo:                   pageRepo,
		pageRevisionRepo:           pageRevisionRepo,
	}
}

// CreateSuggestionInput は編集提案作成の入力パラメータ
type CreateSuggestionInput struct {
	SpaceID          model.SpaceID
	SpaceIdentifier  model.SpaceIdentifier
	TopicID          model.TopicID
	SpaceMemberID    model.SpaceMemberID
	Title            string
	Body             string
	CurrentTopicName string
	DraftPages       []*model.DraftPage
}

// CreateSuggestionOutput は編集提案作成の出力パラメータ
type CreateSuggestionOutput struct {
	Suggestion *model.Suggestion
}

// Execute は編集提案を作成する
func (uc *CreateSuggestionUsecase) Execute(ctx context.Context, input CreateSuggestionInput) (*CreateSuggestionOutput, error) {
	// トランザクション前: 本文HTMLの生成
	bodyHTML, err := uc.renderBodyHTML(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("本文HTMLの生成に失敗しました: %w", err)
	}

	// トランザクション前: 各下書きページの最新リビジョンを取得
	pageRevisions, err := uc.fetchLatestPageRevisions(ctx, input.DraftPages, input.SpaceID)
	if err != nil {
		return nil, err
	}

	// トランザクション: 永続化処理
	return uc.createSuggestion(ctx, input, bodyHTML, pageRevisions)
}

// renderBodyHTML は本文のMarkdownをHTMLに変換し、Wikiリンクを解決する
func (uc *CreateSuggestionUsecase) renderBodyHTML(ctx context.Context, input CreateSuggestionInput) (string, error) {
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

// fetchLatestPageRevisions は各下書きページに対応するページの最新リビジョンを取得する
func (uc *CreateSuggestionUsecase) fetchLatestPageRevisions(ctx context.Context, draftPages []*model.DraftPage, spaceID model.SpaceID) (map[model.PageID]*model.PageRevision, error) {
	pageRevisions := make(map[model.PageID]*model.PageRevision, len(draftPages))

	for _, draftPage := range draftPages {
		latestRevision, err := uc.pageRevisionRepo.FindLatestByPageID(ctx, draftPage.PageID, spaceID)
		if err != nil {
			return nil, fmt.Errorf("ページリビジョンの取得に失敗しました: %w", err)
		}
		if latestRevision == nil {
			return nil, fmt.Errorf("ページ %s のリビジョンが見つかりません", draftPage.PageID)
		}

		pageRevisions[draftPage.PageID] = latestRevision
	}

	return pageRevisions, nil
}

// createSuggestion はトランザクション内で編集提案を作成する
func (uc *CreateSuggestionUsecase) createSuggestion(ctx context.Context, input CreateSuggestionInput, bodyHTML string, pageRevisions map[model.PageID]*model.PageRevision) (*CreateSuggestionOutput, error) {
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
		BodyHTML:             bodyHTML,
		Status:               model.SuggestionStatusOpen,
	})
	if err != nil {
		return nil, fmt.Errorf("編集提案の作成に失敗しました: %w", err)
	}

	// 3. 各下書きページからSuggestionPageとSuggestionPageRevisionを作成
	for _, draftPage := range input.DraftPages {
		latestRevision := pageRevisions[draftPage.PageID]

		// SuggestionPageを作成
		suggestionPage, err := suggestionPageRepo.Create(ctx, repository.CreateSuggestionPageInput{
			SpaceID:                   input.SpaceID,
			SuggestionID:              suggestion.ID,
			PageID:                    draftPage.PageID,
			PageRevisionID:            latestRevision.ID,
			Title:                     draftPage.Title,
			Body:                      draftPage.Body,
			BodyHTML:                  draftPage.BodyHTML,
			LinkedPageIDs:             draftPage.LinkedPageIDs,
			FeaturedImageAttachmentID: draftPage.FeaturedImageAttachmentID,
		})
		if err != nil {
			return nil, fmt.Errorf("編集提案ページの作成に失敗しました: %w", err)
		}

		// SuggestionPageRevisionを作成
		_, err = suggestionPageRevisionRepo.Create(ctx, repository.CreateSuggestionPageRevisionInput{
			SpaceID:             input.SpaceID,
			SuggestionPageID:    suggestionPage.ID,
			EditorSpaceMemberID: input.SpaceMemberID,
			Title:               draftPage.Title,
			Body:                draftPage.Body,
			BodyHTML:            draftPage.BodyHTML,
		})
		if err != nil {
			return nil, fmt.Errorf("編集提案ページリビジョンの作成に失敗しました: %w", err)
		}

		// DraftPageのsuggestion_page_idを設定し、編集提案モードにリンクする
		if draftPage.ID != "" {
			_, err = draftPageRepo.UpdateSuggestionPageID(ctx, draftPage.ID, input.SpaceID, &suggestionPage.ID)
			if err != nil {
				return nil, fmt.Errorf("下書きページのsuggestion_page_id設定に失敗しました: %w", err)
			}
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

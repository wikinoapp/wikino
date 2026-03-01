package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// AutoSaveDraftPageUsecase は下書きページの自動保存ユースケース
type AutoSaveDraftPageUsecase struct {
	db             *sql.DB
	draftPageRepo  *repository.DraftPageRepository
	pageRepo       *repository.PageRepository
	topicRepo      *repository.TopicRepository
	attachmentRepo *repository.AttachmentRepository
}

// NewAutoSaveDraftPageUsecase は AutoSaveDraftPageUsecase を生成する
func NewAutoSaveDraftPageUsecase(
	db *sql.DB,
	draftPageRepo *repository.DraftPageRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	attachmentRepo *repository.AttachmentRepository,
) *AutoSaveDraftPageUsecase {
	return &AutoSaveDraftPageUsecase{
		db:             db,
		draftPageRepo:  draftPageRepo,
		pageRepo:       pageRepo,
		topicRepo:      topicRepo,
		attachmentRepo: attachmentRepo,
	}
}

// AutoSaveDraftPageInput は下書き自動保存の入力パラメータ
type AutoSaveDraftPageInput struct {
	SpaceID          model.SpaceID
	PageID           model.PageID
	SpaceMemberID    model.SpaceMemberID
	TopicID          model.TopicID
	Title            *string
	Body             string
	SpaceIdentifier  model.SpaceIdentifier
	CurrentTopicName string
}

// AutoSaveDraftPageOutput は下書き自動保存の出力パラメータ
type AutoSaveDraftPageOutput struct {
	DraftPage  *model.DraftPage
	ModifiedAt time.Time
}

// findOrCreateRetryLimit はDraftPageのfind_or_create時のリトライ上限
const findOrCreateRetryLimit = 3

// Execute は下書きページを自動保存する
func (uc *AutoSaveDraftPageUsecase) Execute(ctx context.Context, input AutoSaveDraftPageInput) (*AutoSaveDraftPageOutput, error) {
	now := time.Now()

	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	draftPageRepo := uc.draftPageRepo.WithTx(tx)
	pageRepo := uc.pageRepo.WithTx(tx)
	topicRepo := uc.topicRepo.WithTx(tx)
	attachmentRepo := uc.attachmentRepo.WithTx(tx)

	// 1. DraftPageをfind_or_createで取得・作成
	draftPage, err := findOrCreateDraftPage(ctx, draftPageRepo, input, now)
	if err != nil {
		return nil, fmt.Errorf("下書きページの取得・作成に失敗しました: %w", err)
	}

	// 2. Markdownレンダリング
	bodyHTML := markup.RenderMarkdown(input.Body)

	// 3. Wikiリンク解析・リンク先ページの自動作成
	linkedPageIDs, pageLocations, err := resolveAndCreateLinkedPages(
		ctx, input.Body, input.CurrentTopicName, input.SpaceID, pageRepo, topicRepo,
	)
	if err != nil {
		return nil, fmt.Errorf("wikiリンクの解析に失敗しました: %w", err)
	}

	// 4. bodyHTML内のWikiリンクを<a>タグに変換
	if len(pageLocations) > 0 {
		bodyHTML = markup.ReplaceWikilinks(bodyHTML, input.CurrentTopicName, input.SpaceIdentifier, pageLocations)
	}

	// 5. 添付ファイルフィルター
	bodyHTML, err = markup.FilterAttachments(ctx, bodyHTML, input.SpaceID, attachmentRepo)
	if err != nil {
		return nil, fmt.Errorf("添付ファイルのフィルター処理に失敗しました: %w", err)
	}

	// 6. スタンドアロン画像のラッピング
	bodyHTML = markup.WrapStandaloneImageLinks(bodyHTML)

	// 7. DraftPageを更新
	updatedDraftPage, err := draftPageRepo.Update(ctx, repository.UpdateDraftPageInput{
		ID:            draftPage.ID,
		SpaceID:       input.SpaceID,
		TopicID:       input.TopicID,
		Title:         input.Title,
		Body:          input.Body,
		BodyHTML:      bodyHTML,
		LinkedPageIDs: linkedPageIDs,
		ModifiedAt:    now,
	})
	if err != nil {
		return nil, fmt.Errorf("下書きページの更新に失敗しました: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &AutoSaveDraftPageOutput{
		DraftPage:  updatedDraftPage,
		ModifiedAt: now,
	}, nil
}

// findOrCreateDraftPage はDraftPageを取得するか、存在しなければ作成する。
// ユニーク制約（space_member_id + page_id）違反時はリトライする。
func findOrCreateDraftPage(
	ctx context.Context,
	repo *repository.DraftPageRepository,
	input AutoSaveDraftPageInput,
	now time.Time,
) (*model.DraftPage, error) {
	for i := 0; i < findOrCreateRetryLimit; i++ {
		draftPage, err := repo.FindByPageAndMember(ctx, input.PageID, input.SpaceMemberID, input.SpaceID)
		if err != nil {
			return nil, err
		}
		if draftPage != nil {
			return draftPage, nil
		}

		draftPage, err = repo.Create(ctx, repository.CreateDraftPageInput{
			SpaceID:       input.SpaceID,
			PageID:        input.PageID,
			SpaceMemberID: input.SpaceMemberID,
			TopicID:       input.TopicID,
			Title:         input.Title,
			Body:          "",
			BodyHTML:      "",
			LinkedPageIDs: nil,
			ModifiedAt:    now,
		})
		if err != nil {
			if isUniqueViolation(err) {
				slog.WarnContext(ctx, "DraftPageのユニーク制約違反によりリトライ", "attempt", i+1)
				continue
			}
			return nil, err
		}

		return draftPage, nil
	}

	return nil, fmt.Errorf("DraftPageの取得・作成が%d回のリトライ後も失敗しました", findOrCreateRetryLimit)
}

// isUniqueViolation はPostgreSQLのユニーク制約違反エラーかを判定する
func isUniqueViolation(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23505"
	}
	return false
}

// resolveAndCreateLinkedPages はWikiリンクを解析し、リンク先ページを自動作成する
func resolveAndCreateLinkedPages(
	ctx context.Context,
	body string,
	currentTopicName string,
	spaceID model.SpaceID,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
) ([]model.PageID, []markup.PageLocation, error) {
	keys := markup.ScanWikilinks(body, currentTopicName)
	if len(keys) == 0 {
		return nil, nil, nil
	}

	// トピック名の一覧を抽出してバッチ検索
	topicNames := uniqueTopicNames(keys)
	topics, err := topicRepo.FindBySpaceAndNames(ctx, spaceID, topicNames)
	if err != nil {
		return nil, nil, err
	}
	topicMap := make(map[string]*model.Topic, len(topics))
	for _, t := range topics {
		topicMap[t.Name] = t
	}

	var linkedPageIDs []model.PageID
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

		page, err := findOrCreateLinkedPage(ctx, pageRepo, spaceID, topic.ID, key.PageTitle)
		if err != nil {
			return nil, nil, err
		}

		linkedPageIDs = append(linkedPageIDs, page.ID)

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

	return linkedPageIDs, pageLocations, nil
}

// findOrCreateLinkedPage はWikiリンクのリンク先ページを取得するか、存在しなければ作成する。
// ページ番号のユニーク制約（space_id + number）違反時はリトライする。
func findOrCreateLinkedPage(
	ctx context.Context,
	pageRepo *repository.PageRepository,
	spaceID model.SpaceID,
	topicID model.TopicID,
	title string,
) (*model.Page, error) {
	for i := 0; i < findOrCreateRetryLimit; i++ {
		page, err := pageRepo.FindByTopicAndTitle(ctx, topicID, title, spaceID)
		if err != nil {
			return nil, err
		}
		if page != nil {
			return page, nil
		}

		nextNumber, err := pageRepo.NextPageNumber(ctx, spaceID)
		if err != nil {
			return nil, fmt.Errorf("次のページ番号の取得に失敗しました: %w", err)
		}

		page, err = pageRepo.CreateLinkedPage(ctx, repository.CreateLinkedPageInput{
			SpaceID: spaceID,
			TopicID: topicID,
			Number:  nextNumber,
			Title:   title,
		})
		if err != nil {
			if isUniqueViolation(err) {
				slog.WarnContext(ctx, "リンク先ページのユニーク制約違反によりリトライ", "attempt", i+1, "title", title)
				continue
			}
			return nil, fmt.Errorf("リンク先ページの作成に失敗しました: %w", err)
		}

		return page, nil
	}

	return nil, fmt.Errorf("リンク先ページの作成が%d回のリトライ後も失敗しました", findOrCreateRetryLimit)
}

// uniqueTopicNames はWikiリンクキーからユニークなトピック名を抽出する
func uniqueTopicNames(keys []markup.WikilinkKey) []string {
	seen := make(map[string]bool, len(keys))
	var names []string
	for _, key := range keys {
		if !seen[key.TopicName] {
			seen[key.TopicName] = true
			names = append(names, key.TopicName)
		}
	}
	return names
}

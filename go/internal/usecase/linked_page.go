package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// findOrCreateRetryLimitはfind_or_create時のリトライ上限
const findOrCreateRetryLimit = 3

// linkCreatingPageLocationResolverは保存経路 (公開・下書き保存) が使う
// markup.PageLocationResolver。previewPageLocationResolverと異なり、存在しないリンク先
// ページ (とpage_editors) を副作用として自動作成し、得られたリンク先ページIDを記録して
// page / draft_page行に永続化できるようにする。書き込みを行うため、トランザクションに
// バインドしたpageリポジトリで構築し、そのトランザクション内で使う。
type linkCreatingPageLocationResolver struct {
	spaceMemberID  model.SpaceMemberID
	topicRepo      *repository.TopicRepository
	pageRepo       *repository.PageRepository
	pageEditorRepo *repository.PageEditorRepository

	// linkedPageIDsはResolveByKeysで解決・作成されたリンク先ページIDを保持し、
	// 呼び出し元がpage / draft_page行に永続化するために使う。
	linkedPageIDs []model.PageID
}

// ResolveByKeysはWikiリンクキーをページ位置情報に解決し、存在しないページを自動作成する。
func (r *linkCreatingPageLocationResolver) ResolveByKeys(ctx context.Context, keys []markup.WikilinkKey, spaceID model.SpaceID) ([]markup.PageLocation, error) {
	topics, err := r.topicRepo.FindBySpaceAndNames(ctx, spaceID, uniqueTopicNames(keys))
	if err != nil {
		return nil, err
	}
	topicMap := make(map[string]*model.Topic, len(topics))
	for _, t := range topics {
		topicMap[t.Name] = t
	}

	linkedPageIDs, locations, err := resolveAndCreateLinkedPages(
		ctx, keys, topicMap, spaceID, r.spaceMemberID, r.pageRepo, r.pageEditorRepo,
	)
	if err != nil {
		return nil, err
	}
	r.linkedPageIDs = linkedPageIDs
	return locations, nil
}

// resolveAndCreateLinkedPagesは事前に取得したWikiリンクキーとトピックMapを使い、リンク先ページを自動作成する
func resolveAndCreateLinkedPages(
	ctx context.Context,
	keys []markup.WikilinkKey,
	topicMap map[string]*model.Topic,
	spaceID model.SpaceID,
	spaceMemberID model.SpaceMemberID,
	pageRepo *repository.PageRepository,
	pageEditorRepo *repository.PageEditorRepository,
) ([]model.PageID, []markup.PageLocation, error) {
	if len(keys) == 0 {
		return nil, nil, nil
	}

	now := time.Now()
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

		page, created, err := findOrCreateLinkedPage(ctx, pageRepo, spaceID, topic.ID, key.PageTitle)
		if err != nil {
			return nil, nil, err
		}

		// Rails版と同様に、ページを新規作成した場合はpage_editorsレコードも作成する
		if created {
			_, err = pageEditorRepo.FindOrCreate(ctx, repository.FindOrCreateInput{
				SpaceID:            spaceID,
				PageID:             page.ID,
				SpaceMemberID:      spaceMemberID,
				LastPageModifiedAt: now,
			})
			if err != nil {
				return nil, nil, fmt.Errorf("自動作成ページの編集者登録に失敗しました: %w", err)
			}
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

// findOrCreateLinkedPageはWikiリンクのリンク先ページを取得するか、存在しなければ作成する。
// ページ番号のユニーク制約 (space_id + number) 違反時はリトライする。
// 戻り値のboolはページが新規作成された場合にtrueを返す。
func findOrCreateLinkedPage(
	ctx context.Context,
	pageRepo *repository.PageRepository,
	spaceID model.SpaceID,
	topicID model.TopicID,
	title string,
) (*model.Page, bool, error) {
	for i := 0; i < findOrCreateRetryLimit; i++ {
		page, err := pageRepo.FindByTopicAndTitle(ctx, topicID, title, spaceID)
		if err != nil {
			return nil, false, err
		}
		if page != nil {
			return page, false, nil
		}

		nextNumber, err := pageRepo.NextPageNumber(ctx, spaceID)
		if err != nil {
			return nil, false, fmt.Errorf("次のページ番号の取得に失敗しました: %w", err)
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
			return nil, false, fmt.Errorf("リンク先ページの作成に失敗しました: %w", err)
		}

		return page, true, nil
	}

	return nil, false, fmt.Errorf("リンク先ページの作成が%d回のリトライ後も失敗しました", findOrCreateRetryLimit)
}

// uniqueTopicNamesはWikiリンクキーからユニークなトピック名を抽出する
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

// isUniqueViolationはPostgreSQLのユニーク制約違反エラーかを判定する
func isUniqueViolation(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23505"
	}
	return false
}

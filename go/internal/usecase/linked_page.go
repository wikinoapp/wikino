package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// linkCreatingPageLocationResolverは保存経路 (公開・下書き保存) が使う
// markup.PageLocationResolver。previewPageLocationResolverと異なり、存在しないリンク先
// ページ (とpage_editors) を副作用として自動作成し、得られたリンク先ページIDを記録して
// page / draft_page行に永続化できるようにする。書き込みを行うため、トランザクションに
// バインドしたpageリポジトリで構築し、そのトランザクション内で使う。
type linkCreatingPageLocationResolver struct {
	spaceMemberID model.SpaceMemberID
	// canCreateInはリンク先ページを作れるトピックかを返す。nilならすべてのトピックに作る
	canCreateIn    func(topic *model.Topic) bool
	topicRepo      *repository.TopicRepository
	pageRepo       *repository.PageRepository
	pageEditorRepo *repository.PageEditorRepository

	// linkedPageIDsはResolveByKeysで解決・作成されたリンク先ページIDを保持し、
	// 呼び出し元がpage / draft_page行に永続化するために使う。
	linkedPageIDs []model.PageID
}

// ResolveByKeysはWikiリンクキーをページ位置情報に解決し、存在しないページを自動作成する。
func (r *linkCreatingPageLocationResolver) ResolveByKeys(ctx context.Context, keys []markup.WikilinkKey, spaceID model.SpaceID) ([]markup.PageLocation, error) {
	targets, err := findWikilinkTargets(ctx, keys, spaceID, r.topicRepo, r.pageRepo)
	if err != nil {
		return nil, err
	}

	linkedPageIDs, locations, err := resolveAndCreateLinkedPages(
		ctx, targets, spaceID, r.spaceMemberID, r.canCreateIn, r.pageRepo, r.pageEditorRepo,
	)
	if err != nil {
		return nil, err
	}
	r.linkedPageIDs = linkedPageIDs
	return locations, nil
}

// resolveLinkedPageIDsInputはリンク先ページIDの解決に必要なパラメータ
type resolveLinkedPageIDsInput struct {
	Body             string
	CurrentTopicName string
	SpaceID          model.SpaceID
	SpaceMemberID    model.SpaceMemberID
	// CanCreateLinkedPageInはリンク先ページを作れるトピックかを返す。nilならすべてのトピックに作る (Web画面)。
	// 公開APIはトークンで見られないトピックにページを作らないよう、見られるかの判定を渡す
	// (Wikiリンクのプレビューと同じ判定にし、プレビューと書き込みの結果を揃える)
	CanCreateLinkedPageIn func(topic *model.Topic) bool
}

// resolveLinkedPageIDsは本文のWikiリンクが指すページIDを現れる順に返し、存在しないリンク先
// ページを自動作成する。本文HTMLを保存しない保存経路で、ScanWikilinksが抽出したキーを
// linkCreatingPageLocationResolverに解決させる。画面と同じリンク集合を読むため、
// ScanWikilinks内部ではHTMLの描画・サニタイズを行う。
// 書き込みを行うため、トランザクションにバインドしたリポジトリを渡すこと。
func resolveLinkedPageIDs(
	ctx context.Context,
	input resolveLinkedPageIDsInput,
	topicRepo *repository.TopicRepository,
	pageRepo *repository.PageRepository,
	pageEditorRepo *repository.PageEditorRepository,
) ([]model.PageID, error) {
	keys := markup.ScanWikilinks(input.Body, input.CurrentTopicName)
	if len(keys) == 0 {
		return nil, nil
	}

	resolver := &linkCreatingPageLocationResolver{
		spaceMemberID:  input.SpaceMemberID,
		canCreateIn:    input.CanCreateLinkedPageIn,
		topicRepo:      topicRepo,
		pageRepo:       pageRepo,
		pageEditorRepo: pageEditorRepo,
	}
	if _, err := resolver.ResolveByKeys(ctx, keys, input.SpaceID); err != nil {
		return nil, err
	}

	return resolver.linkedPageIDs, nil
}

// apiLinkedPageTopicFilterは、公開APIの書き込みでWikiリンク先のページを作れるトピックかを返す関数を作る。
// トークンで見られないトピックにはページを作らない。見られないトピックの存在とページの有無を明かさないよう、
// Wikiリンクのプレビューはそのトピックへのリンクを「トピックが無い」と示すため、書き込みもそれに揃える。
// 既存のページへのリンクは、Web画面で保存したときと同じリンク一覧になるよう記録する
func apiLinkedPageTopicFilter(
	ctx context.Context,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	principal *model.APIPrincipal,
) (func(topic *model.Topic) bool, error) {
	access, err := fetchAPITopicAccess(ctx, topicRepo, topicMemberRepo, principal)
	if err != nil {
		return nil, fmt.Errorf("トピックの閲覧権限の取得に失敗: %w", err)
	}
	return access.canShowTopic, nil
}

// wikilinkTargetはWikiリンクが指すトピックと既存のページを、ページの作成を伴わずに解決した結果
type wikilinkTarget struct {
	key markup.WikilinkKey
	// topicはリンク先のトピック (廃棄済みを除く)。トピックが無ければnil
	topic *model.Topic
	// pageはリンク先の既存のページ (廃棄済み・ゴミ箱のページを含む)。トピックかページが無ければnil
	page *repository.LinkedPage
}

// findWikilinkTargetsはWikiリンクキーを重複を除いて現れる順に並べ、それぞれのリンク先のトピックと
// 既存のページを解決する。書き込みを行わない読み取りの関数で、保存経路 (resolveAndCreateLinkedPages) と
// Wikiリンクのプレビューが共有し、どのリンク先が既存のページになり、どれが作られるかの判定を揃える。
// トピックとページはそれぞれ1クエリでまとめて取得する
func findWikilinkTargets(
	ctx context.Context,
	keys []markup.WikilinkKey,
	spaceID model.SpaceID,
	topicRepo *repository.TopicRepository,
	pageRepo *repository.PageRepository,
) ([]wikilinkTarget, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	topics, err := topicRepo.FindBySpaceAndNames(ctx, spaceID, uniqueTopicNames(keys))
	if err != nil {
		return nil, err
	}
	topicMap := make(map[string]*model.Topic, len(topics))
	for _, t := range topics {
		topicMap[t.Name] = t
	}

	existingPages, err := findPagesByWikilinkKeys(ctx, pageRepo, keys, topicMap, spaceID)
	if err != nil {
		return nil, err
	}

	targets := make([]wikilinkTarget, 0, len(keys))
	seen := make(map[string]bool, len(keys))
	for _, key := range keys {
		lookupKey := key.TopicName + "/" + key.PageTitle
		if seen[lookupKey] {
			continue
		}
		seen[lookupKey] = true

		target := wikilinkTarget{key: key, topic: topicMap[key.TopicName]}
		if target.topic != nil {
			target.page = existingPages[repository.TopicPageTitle{TopicID: target.topic.ID, Title: key.PageTitle}]
		}
		targets = append(targets, target)
	}

	return targets, nil
}

// resolveAndCreateLinkedPagesはfindWikilinkTargetsで解決したリンク先のうち、ページが無いものを自動作成する。
// トピックが無いリンク先と、canCreateInがfalseを返すトピックの存在しないページはリンクにしない
func resolveAndCreateLinkedPages(
	ctx context.Context,
	targets []wikilinkTarget,
	spaceID model.SpaceID,
	spaceMemberID model.SpaceMemberID,
	canCreateIn func(topic *model.Topic) bool,
	pageRepo *repository.PageRepository,
	pageEditorRepo *repository.PageEditorRepository,
) ([]model.PageID, []markup.PageLocation, error) {
	now := time.Now()
	var linkedPageIDs []model.PageID
	var pageLocations []markup.PageLocation

	for _, target := range targets {
		if target.topic == nil {
			continue
		}

		page := target.page
		created := false
		if page == nil {
			if canCreateIn != nil && !canCreateIn(target.topic) {
				continue
			}

			createdPage, createdNow, err := findOrCreateLinkedPage(ctx, pageRepo, spaceID, target.topic.ID, target.key.PageTitle)
			if err != nil {
				return nil, nil, err
			}
			// 一括取得と同じ射影に揃え、この先の組み立てが取得経路を意識せずに済むようにする。
			page = &repository.LinkedPage{
				ID:     createdPage.ID,
				Number: createdPage.Number,
				Title:  createdPage.Title,
			}
			created = createdNow
		}

		// Rails版と同様に、ページを新規作成した場合はpage_editorsレコードも作成する
		if created {
			_, err := pageEditorRepo.FindOrCreate(ctx, repository.FindOrCreateInput{
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

		pageTitle := target.key.PageTitle
		if page.Title != nil {
			pageTitle = *page.Title
		}
		pageLocations = append(pageLocations, markup.PageLocation{
			Key:        target.key,
			TopicName:  target.topic.Name,
			PageID:     page.ID,
			PageNumber: int(page.Number),
			PageTitle:  pageTitle,
		})
	}

	return linkedPageIDs, pageLocations, nil
}

// findOrCreateLinkedPageはWikiリンクのリンク先ページを取得するか、存在しなければ作成する。
// 呼び出し元のトランザクション内で一意制約に違反した場合は、トランザクション全体の
// やり直しが必要なため、元のエラーを呼び出し元へ返す。
// 戻り値のboolはページが新規作成された場合にtrueを返す。
func findOrCreateLinkedPage(
	ctx context.Context,
	pageRepo *repository.PageRepository,
	spaceID model.SpaceID,
	topicID model.TopicID,
	title string,
) (*model.Page, bool, error) {
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
		return nil, false, fmt.Errorf("リンク先ページの作成に失敗しました: %w", err)
	}

	return page, true, nil
}

// findPagesByWikilinkKeysはWikiリンクキーが指す既存ページを1クエリで取得する。
// DBの照合結果を保持するため、入力のトピックIDとタイトルをそのままmapのキーにする。
func findPagesByWikilinkKeys(
	ctx context.Context,
	pageRepo *repository.PageRepository,
	keys []markup.WikilinkKey,
	topicMap map[string]*model.Topic,
	spaceID model.SpaceID,
) (map[repository.TopicPageTitle]*repository.LinkedPage, error) {
	pairs := make([]repository.TopicPageTitle, 0, len(keys))
	seen := make(map[repository.TopicPageTitle]bool, len(keys))
	for _, key := range keys {
		topic := topicMap[key.TopicName]
		if topic == nil {
			continue
		}

		lookupKey := repository.TopicPageTitle{TopicID: topic.ID, Title: key.PageTitle}
		if seen[lookupKey] {
			continue
		}
		seen[lookupKey] = true

		pairs = append(pairs, lookupKey)
	}

	return pageRepo.FindByTopicAndTitlePairs(ctx, pairs, spaceID)
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

// isUniqueViolationOnはconstraintの一意制約に違反したエラーかを判定する
func isUniqueViolationOn(err error, constraint string) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505" && pqErr.Constraint == constraint
}

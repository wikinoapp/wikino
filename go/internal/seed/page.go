package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// seededPageは作成したページ1件。numberはページのURLを組み立てる元に
// なり、idは後続の生成器がWikiリンクの向き先として使う。
type seededPage struct {
	id     model.PageID
	number model.PageNumber
	title  string
}

// pageWriterは1つのスペースにページを作成する。関数ではなく値にして
// いるのは、必要なRepositoryを一度だけ組み立ててすべてのページで使い回す
// ため。後続の生成器はページを数百件作る。
type pageWriter struct {
	dbtx             query.DBTX
	space            *seededSpace
	topicRepo        *repository.TopicRepository
	pageRepo         *repository.PageRepository
	pageRevisionRepo *repository.PageRevisionRepository
	pageEditorRepo   *repository.PageEditorRepository
	attachmentRepo   *repository.AttachmentRepository
}

// newPageWriterはspaceにページを作成するwriterを返す。
func newPageWriter(dbtx query.DBTX, space *seededSpace) *pageWriter {
	queries := query.New(dbtx)

	return &pageWriter{
		dbtx:             dbtx,
		space:            space,
		topicRepo:        repository.NewTopicRepository(queries),
		pageRepo:         repository.NewPageRepository(queries),
		pageRevisionRepo: repository.NewPageRevisionRepository(queries),
		pageEditorRepo:   repository.NewPageEditorRepository(queries),
		attachmentRepo:   repository.NewAttachmentRepository(queries),
	}
}

// createPageInputは作成するページ1件の内容。
type createPageInput struct {
	topic *seededTopic
	// authorはページの書き手として記録するスペースメンバー。ページ詳細
	// 画面に出る名前と、ページが並ぶホーム画面の持ち主を決める。
	author *seededSpaceMember
	title  string
	body   string
}

// createPageは公開済みのページ1件と、その最初のリビジョン、編集者
// エントリを作成する。これは画面から公開したときに残る行の一式にあたる。
//
// ページ行をRepositoryではなくここで書くのは、ページの作成を担当しているのが
// 今もRails側であり、Go側にはUpdateしか無いため。リビジョンと編集者
// エントリには本番のCreateがあるので、そちらはRepositoryを経由する。
func (w *pageWriter) createPage(ctx context.Context, input createPageInput) (*seededPage, error) {
	if err := w.ensureTopicInSpace(input.topic); err != nil {
		return nil, err
	}

	// 先にレンダリングする。resolverは本文がリンクする先のページを作成し、
	// それらがページ番号を消費するため。先にこのページの番号を取ると、リンク先が
	// あとからその番号を取ってしまう。
	bodyHTML, linkedPageIDs, err := w.render(ctx, input)
	if err != nil {
		return nil, err
	}

	number, err := w.pageRepo.NextPageNumber(ctx, w.space.id)
	if err != nil {
		return nil, fmt.Errorf("次のページ番号の取得に失敗: %w", err)
	}

	now := time.Now()

	var id string
	err = w.dbtx.QueryRowContext(
		ctx,
		`INSERT INTO pages
           (space_id, topic_id, number, title, body, body_html, linked_page_ids,
            modified_at, published_at, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8, $8, $8)
         RETURNING id`,
		string(w.space.id), string(input.topic.id), int32(number), input.title,
		input.body, bodyHTML, pq.Array(pageIDStrings(linkedPageIDs)), now,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("ページ %sの作成に失敗: %w", input.title, err)
	}

	pageID := model.PageID(id)

	if _, err := w.pageRevisionRepo.Create(ctx, repository.CreatePageRevisionInput{
		SpaceID:       w.space.id,
		SpaceMemberID: input.author.id,
		PageID:        pageID,
		Title:         input.title,
		Body:          input.body,
		BodyHTML:      bodyHTML,
	}); err != nil {
		return nil, fmt.Errorf("ページ %sのリビジョンの作成に失敗: %w", input.title, err)
	}

	if _, err := w.pageEditorRepo.FindOrCreate(ctx, repository.FindOrCreateInput{
		SpaceID:            w.space.id,
		PageID:             pageID,
		SpaceMemberID:      input.author.id,
		LastPageModifiedAt: now,
	}); err != nil {
		return nil, fmt.Errorf("ページ %sの編集者の登録に失敗: %w", input.title, err)
	}

	return &seededPage{id: pageID, number: number, title: input.title}, nil
}

// ensureTopicInSpaceは、別のスペースに属するトピックを拒否する。
//
// スペースはwriterから、トピックは呼び出し元から来る。pagesには
// (space_id, topic_id) の複合外部キーが無いため、食い違ったまま書けてしまい、
// さらに本文のWikiリンクにも別のスペースの識別子が入ってしまう。
func (w *pageWriter) ensureTopicInSpace(topic *seededTopic) error {
	if topic.spaceID != w.space.id {
		return fmt.Errorf("トピック %sはスペース %sに属していない", topic.name, w.space.identifier)
	}

	return nil
}

// renderはMarkdown本文を、ページ詳細画面が配信するHTMLに変換し、
// 本文がリンクする先のページを併せて返す。
func (w *pageWriter) render(ctx context.Context, input createPageInput) (string, []model.PageID, error) {
	resolver := &seedPageLocationResolver{
		author:         input.author,
		topicRepo:      w.topicRepo,
		pageRepo:       w.pageRepo,
		pageEditorRepo: w.pageEditorRepo,
	}

	bodyHTML, err := markup.RenderHTML(
		ctx, input.body, input.topic.name, w.space.id, w.space.identifier, resolver, w.attachmentRepo,
	)
	if err != nil {
		return "", nil, fmt.Errorf("ページ %sの本文のレンダリングに失敗: %w", input.title, err)
	}

	return bodyHTML, resolver.linkedPageIDs, nil
}

// seedPageLocationResolverは本文のWikiリンクを解決し、リンク先のページを
// 作成する。画面から公開したときと同じ振る舞い。
//
// 本番のresolverはusecaseパッケージに非公開で置かれている。公開すると、
// シードのために本番のAPIを広げることになり、RepositoryへCreateを足さない
// ことで避けているのと同じ取引になる。ここで重複するのは小さな範囲であり、
// シードには不要なものを落とせる。一意制約違反時のリトライは2つの書き手が同じ
// ページ番号を取り合う状況への備えであり、シードは空にしたばかりの
// データベースに対する唯一の書き手であるため。
type seedPageLocationResolver struct {
	author         *seededSpaceMember
	topicRepo      *repository.TopicRepository
	pageRepo       *repository.PageRepository
	pageEditorRepo *repository.PageEditorRepository

	// linkedPageIDsは本文のリンク先を集め、呼び出し元がページ行へ保存する。
	// バックリンクはこの列から引かれるため、ここを書くことでリンクが両方向から
	// 見えるようになる。
	linkedPageIDs []model.PageID
}

// ResolveByKeysはWikiリンクキーをページ位置情報へ解決し、まだ存在しない
// ページを作成する。
func (r *seedPageLocationResolver) ResolveByKeys(
	ctx context.Context,
	keys []markup.WikilinkKey,
	spaceID model.SpaceID,
) ([]markup.PageLocation, error) {
	topics, err := r.topicRepo.FindBySpaceAndNames(ctx, spaceID, uniqueTopicNames(keys))
	if err != nil {
		return nil, fmt.Errorf("リンク先トピックの取得に失敗: %w", err)
	}
	topicsByName := make(map[string]*model.Topic, len(topics))
	for _, topic := range topics {
		topicsByName[topic.Name] = topic
	}

	locations := make([]markup.PageLocation, 0, len(keys))
	seen := make(map[string]bool, len(keys))

	for _, key := range keys {
		// 同じページを指すキーが2つあると2回作成してしまい、
		// (topic_id, title) の一意インデックスで衝突するため。
		lookup := key.TopicName + "/" + key.PageTitle
		if seen[lookup] {
			continue
		}
		seen[lookup] = true

		// 存在しないトピックへのリンクは、本番と同じくプレーンテキストの
		// ままになる。シードはそうしたリンクを意図的に書き、未解決の見た目も
		// 画面で確認できるようにしている。
		topic := topicsByName[key.TopicName]
		if topic == nil {
			continue
		}

		page, err := r.findOrCreatePage(ctx, spaceID, topic.ID, key.PageTitle)
		if err != nil {
			return nil, err
		}
		r.linkedPageIDs = append(r.linkedPageIDs, page.ID)

		title := key.PageTitle
		if page.Title != nil {
			title = *page.Title
		}
		locations = append(locations, markup.PageLocation{
			Key:        key,
			TopicName:  topic.Name,
			PageID:     page.ID,
			PageNumber: int(page.Number),
			PageTitle:  title,
		})
	}

	return locations, nil
}

// findOrCreatePageはリンク先のページを返す。Wikiリンクの指すタイトルが
// まだ書かれていない場合は、未公開のページとして作成する。
func (r *seedPageLocationResolver) findOrCreatePage(
	ctx context.Context,
	spaceID model.SpaceID,
	topicID model.TopicID,
	title string,
) (*model.Page, error) {
	page, err := r.pageRepo.FindByTopicAndTitle(ctx, topicID, title, spaceID)
	if err != nil {
		return nil, fmt.Errorf("リンク先ページ %sの取得に失敗: %w", title, err)
	}
	if page != nil {
		return page, nil
	}

	number, err := r.pageRepo.NextPageNumber(ctx, spaceID)
	if err != nil {
		return nil, fmt.Errorf("次のページ番号の取得に失敗: %w", err)
	}

	page, err = r.pageRepo.CreateLinkedPage(ctx, repository.CreateLinkedPageInput{
		SpaceID: spaceID,
		TopicID: topicID,
		Number:  number,
		Title:   title,
	})
	if err != nil {
		return nil, fmt.Errorf("リンク先ページ %sの作成に失敗: %w", title, err)
	}

	if _, err := r.pageEditorRepo.FindOrCreate(ctx, repository.FindOrCreateInput{
		SpaceID:            spaceID,
		PageID:             page.ID,
		SpaceMemberID:      r.author.id,
		LastPageModifiedAt: time.Now(),
	}); err != nil {
		return nil, fmt.Errorf("リンク先ページ %sの編集者の登録に失敗: %w", title, err)
	}

	return page, nil
}

// uniqueTopicNamesは、キーが名指しするトピックを重複なく列挙する。
func uniqueTopicNames(keys []markup.WikilinkKey) []string {
	seen := make(map[string]bool, len(keys))
	names := make([]string, 0, len(keys))
	for _, key := range keys {
		if seen[key.TopicName] {
			continue
		}
		seen[key.TopicName] = true
		names = append(names, key.TopicName)
	}

	return names
}

// pageIDStringsはページIDを保存用に変換する。結果がnilになることは
// ない。linked_page_ids列はNOT NULLであり、pqはnilのスライスを、リンクの
// 無いページに必要な空配列ではなくNULLとして送るため。
func pageIDStrings(ids []model.PageID) []string {
	ss := make([]string, 0, len(ids))
	for _, id := range ids {
		ss = append(ss, string(id))
	}

	return ss
}

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

// seededPage is one created page. The number is what the page's URL is built
// from, and the id is what a later generator points a wiki link at.
//
// [Ja] seededPage は作成したページ 1 件。number はページの URL を組み立てる元に
// なり、id は後続の生成器が Wiki リンクの向き先として使う。
type seededPage struct {
	id     model.PageID
	number model.PageNumber
	title  string
}

// pageWriter creates pages in one space. It exists as a value rather than as a
// bare function because the repositories it needs are built once and then used
// by every page: later generators create pages in the hundreds.
//
// [Ja] pageWriter は 1 つのスペースにページを作成する。関数ではなく値にして
// いるのは、必要な Repository を一度だけ組み立ててすべてのページで使い回す
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

// newPageWriter returns a writer that creates pages in space.
//
// [Ja] newPageWriter は space にページを作成する writer を返す。
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

// createPageInput describes one page to create.
//
// [Ja] createPageInput は作成するページ 1 件の内容。
type createPageInput struct {
	topic *seededTopic
	// author is the space member the page is attributed to. It decides whose
	// name the page detail screen shows and whose home screen lists the page.
	//
	// [Ja] author はページの書き手として記録するスペースメンバー。ページ詳細
	// 画面に出る名前と、ページが並ぶホーム画面の持ち主を決める。
	author *seededSpaceMember
	title  string
	body   string
}

// createPage creates one published page, its first revision and its editor
// entry, which is the set of rows publishing from the screen leaves behind.
//
// The page row is written here rather than through a repository because
// creating a page is still handled by the Rails side, and the Go side has only
// an Update to call. The revision and the editor entry do have a production
// Create, so those go through their repositories.
//
// [Ja] createPage は公開済みのページ 1 件と、その最初のリビジョン、編集者
// エントリを作成する。これは画面から公開したときに残る行の一式にあたる。
//
// ページ行を Repository ではなくここで書くのは、ページの作成を担当しているのが
// 今も Rails 側であり、Go 側には Update しか無いため。リビジョンと編集者
// エントリには本番の Create があるので、そちらは Repository を経由する。
func (w *pageWriter) createPage(ctx context.Context, input createPageInput) (*seededPage, error) {
	if err := w.ensureTopicInSpace(input.topic); err != nil {
		return nil, err
	}

	// Render first: the resolver creates the pages the body links to, and
	// those take page numbers. Asking for this page's number before that would
	// hand out a number the linked pages then take.
	//
	// [Ja] 先にレンダリングする。resolver は本文がリンクする先のページを作成し、
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
		return nil, fmt.Errorf("ページ %s の作成に失敗: %w", input.title, err)
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
		return nil, fmt.Errorf("ページ %s のリビジョンの作成に失敗: %w", input.title, err)
	}

	if _, err := w.pageEditorRepo.FindOrCreate(ctx, repository.FindOrCreateInput{
		SpaceID:            w.space.id,
		PageID:             pageID,
		SpaceMemberID:      input.author.id,
		LastPageModifiedAt: now,
	}); err != nil {
		return nil, fmt.Errorf("ページ %s の編集者の登録に失敗: %w", input.title, err)
	}

	return &seededPage{id: pageID, number: number, title: input.title}, nil
}

// ensureTopicInSpace rejects a topic that belongs to another space.
//
// The space comes from the writer while the topic comes from the caller, and
// pages has no composite foreign key over (space_id, topic_id). A mismatch
// would be written without complaint, and the body's wiki links would be given
// the other space's identifier on top of that.
//
// [Ja] ensureTopicInSpace は、別のスペースに属するトピックを拒否する。
//
// スペースは writer から、トピックは呼び出し元から来る。pages には
// (space_id, topic_id) の複合外部キーが無いため、食い違ったまま書けてしまい、
// さらに本文の Wiki リンクにも別のスペースの識別子が入ってしまう。
func (w *pageWriter) ensureTopicInSpace(topic *seededTopic) error {
	if topic.spaceID != w.space.id {
		return fmt.Errorf("トピック %s はスペース %s に属していない", topic.name, w.space.identifier)
	}

	return nil
}

// render turns the Markdown body into the HTML the page detail screen serves,
// and reports the pages the body links to.
//
// [Ja] render は Markdown 本文を、ページ詳細画面が配信する HTML に変換し、
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
		return "", nil, fmt.Errorf("ページ %s の本文のレンダリングに失敗: %w", input.title, err)
	}

	return bodyHTML, resolver.linkedPageIDs, nil
}

// seedPageLocationResolver resolves the wiki links of a body, creating the
// pages they point at, the way publishing from the screen does.
//
// The production resolver lives unexported in the usecase package. Exporting
// it would widen a production API for the seed's sake, which is the same trade
// the seed avoids by not adding Create methods to the repositories. What is
// duplicated here is small, and the seed can drop what it does not need: the
// retry on a unique violation guards against two writers racing for the same
// page number, and the seed is one writer against a database it has just
// emptied.
//
// [Ja] seedPageLocationResolver は本文の Wiki リンクを解決し、リンク先のページを
// 作成する。画面から公開したときと同じ振る舞い。
//
// 本番の resolver は usecase パッケージに非公開で置かれている。公開すると、
// シードのために本番の API を広げることになり、Repository へ Create を足さない
// ことで避けているのと同じ取引になる。ここで重複するのは小さな範囲であり、
// シードには不要なものを落とせる。一意制約違反時のリトライは 2 つの書き手が同じ
// ページ番号を取り合う状況への備えであり、シードは空にしたばかりの
// データベースに対する唯一の書き手であるため。
type seedPageLocationResolver struct {
	author         *seededSpaceMember
	topicRepo      *repository.TopicRepository
	pageRepo       *repository.PageRepository
	pageEditorRepo *repository.PageEditorRepository

	// linkedPageIDs collects what the body links to, for the caller to store
	// on the page row. Backlinks are read from that column, so writing it is
	// what makes the link show up from both ends.
	//
	// [Ja] linkedPageIDs は本文のリンク先を集め、呼び出し元がページ行へ保存する。
	// バックリンクはこの列から引かれるため、ここを書くことでリンクが両方向から
	// 見えるようになる。
	linkedPageIDs []model.PageID
}

// ResolveByKeys resolves wiki link keys to page locations, creating the pages
// that do not exist yet.
//
// [Ja] ResolveByKeys は Wiki リンクキーをページ位置情報へ解決し、まだ存在しない
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
		// Two keys naming the same page would otherwise be created twice and
		// collide on the unique index over (topic_id, title).
		//
		// [Ja] 同じページを指すキーが 2 つあると 2 回作成してしまい、
		// (topic_id, title) の一意インデックスで衝突するため。
		lookup := key.TopicName + "/" + key.PageTitle
		if seen[lookup] {
			continue
		}
		seen[lookup] = true

		// A link into a topic that does not exist stays plain text, as it does
		// in production. The seed writes such links on purpose, so that the
		// unresolved form can be seen on the screen too.
		//
		// [Ja] 存在しないトピックへのリンクは、本番と同じくプレーンテキストの
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

// findOrCreatePage returns the linked page, creating it as an unpublished page
// if the wiki link points at a title nothing has written yet.
//
// [Ja] findOrCreatePage はリンク先のページを返す。Wiki リンクの指すタイトルが
// まだ書かれていない場合は、未公開のページとして作成する。
func (r *seedPageLocationResolver) findOrCreatePage(
	ctx context.Context,
	spaceID model.SpaceID,
	topicID model.TopicID,
	title string,
) (*model.Page, error) {
	page, err := r.pageRepo.FindByTopicAndTitle(ctx, topicID, title, spaceID)
	if err != nil {
		return nil, fmt.Errorf("リンク先ページ %s の取得に失敗: %w", title, err)
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
		return nil, fmt.Errorf("リンク先ページ %s の作成に失敗: %w", title, err)
	}

	if _, err := r.pageEditorRepo.FindOrCreate(ctx, repository.FindOrCreateInput{
		SpaceID:            spaceID,
		PageID:             page.ID,
		SpaceMemberID:      r.author.id,
		LastPageModifiedAt: time.Now(),
	}); err != nil {
		return nil, fmt.Errorf("リンク先ページ %s の編集者の登録に失敗: %w", title, err)
	}

	return page, nil
}

// uniqueTopicNames lists the topics the keys name, without repeats.
//
// [Ja] uniqueTopicNames は、キーが名指しするトピックを重複なく列挙する。
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

// pageIDStrings converts page ids for storage. The result is never nil: the
// linked_page_ids column is NOT NULL, and pq sends a nil slice as NULL rather
// than as the empty array a page without links needs.
//
// [Ja] pageIDStrings はページ ID を保存用に変換する。結果が nil になることは
// ない。linked_page_ids 列は NOT NULL であり、pq は nil のスライスを、リンクの
// 無いページに必要な空配列ではなく NULL として送るため。
func pageIDStrings(ids []model.PageID) []string {
	ss := make([]string, 0, len(ids))
	for _, id := range ids {
		ss = append(ss, string(id))
	}

	return ss
}

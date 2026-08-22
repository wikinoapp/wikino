package seed

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// Titles of the drafts that hang on pages nothing has ever published. They are
// named rather than numbered because each one is there for a state of its own,
// and because the draft listings show a draft under its own title.
//
// [Ja] 一度も公開されていないページに付く下書きのタイトル。番号ではなく名前を
// 付けているのは、それぞれが固有の状態を見せるために存在することと、下書きの
// 一覧が下書き自身のタイトルで表示することによる。
const (
	unpublishedDraftTitle = "未公開ページの下書き"
	longHistoryDraftTitle = "履歴の長い下書き"
)

// ordinaryDraftRevisions is how many times a draft has been saved unless it was
// written to carry a history. Two is the smallest number that gives the edit
// history something to compare: a draft only ever comes into being through a
// save, so one revision is the least it can have, and the diff of a revision is
// taken against the one before it.
//
// [Ja] ordinaryDraftRevisions は、履歴を持たせる目的で書いたもの以外の下書きが
// 保存された回数。2 は編集履歴に比較対象を与える最小の数である。下書きは保存に
// よってしか生まれないためリビジョンは最低 1 件あり、リビジョンの差分はその 1 つ
// 前との間で取られるため。
const ordinaryDraftRevisions = 2

// draftModifiedAtStep is the gap the seed leaves between the modified_at
// stamps of two drafts. Both listings that show drafts order them by
// modified_at descending, and stamping several of them with the same instant
// would leave their order to the ids, which would change which drafts the home
// screen keeps from run to run.
//
// [Ja] draftModifiedAtStep は、シードが 2 件の下書きの modified_at の打刻の間に
// 空ける間隔。下書きを見せる 2 つの一覧はいずれも modified_at の降順で並べるため、
// 複数の下書きを同じ時刻で打刻すると並び順が ID 任せになり、ホーム画面が残す下書きが
// 実行ごとに変わってしまう。
const draftModifiedAtStep = time.Minute

// draftStamps hands out the modified_at stamps of every draft a run writes.
// Each draft takes the next stamp, counting back from the instant the run
// started, so the drafts stand in the listings in the order the run created
// them: the first one created is the most recently modified.
//
// One counter serves the whole run rather than one per phase. The drafts on
// unpublished pages are created first so that the home screen, which keeps only
// the newest few, is where they are met. A later phase stamping from its own
// time.Now() would land ahead of them and push them off that screen, because it
// runs after the phase that wrote them.
//
// [Ja] draftStamps は、実行 1 回が書くすべての下書きの modified_at の打刻を渡す。
// 各下書きは次の打刻を受け取り、実行を開始した時刻から遡っていくため、下書きは実行が
// 作成した順に一覧へ並ぶ。最初に作成したものが、最終更新の最も新しいものになる。
//
// カウンターはフェーズごとではなく実行全体で 1 つとする。未公開のページに付く下書きを
// 最初に作成しているのは、新しいものを数件しか残さないホーム画面がそれらと出会う場に
// なるようにするためである。後続のフェーズがそれぞれの time.Now() で打刻すると、
// 下書きを書いたフェーズより後に走る分だけそれらより新しくなり、その画面から
// 押し出してしまう。
type draftStamps struct {
	origin time.Time
	issued int
}

// newDraftStamps returns stamps counting back from origin.
//
// [Ja] newDraftStamps は origin から遡っていく打刻を返す。
func newDraftStamps(origin time.Time) *draftStamps {
	return &draftStamps{origin: origin}
}

// next returns the stamp for the next draft to be created.
//
// [Ja] next は、次に作成する下書きの打刻を返す。
func (s *draftStamps) next() time.Time {
	at := s.origin.Add(-time.Duration(s.issued) * draftModifiedAtStep)
	s.issued++

	return at
}

// newPageDraftSpec describes one draft written against a page that has never
// been published, which is what the editor leaves behind between pressing the
// new page button and publishing.
//
// [Ja] newPageDraftSpec は、一度も公開されていないページに対して書かれた下書き
// 1 件の内容。これは、ページ作成ボタンを押してから公開するまでの間に編集画面が
// 残すものにあたる。
type newPageDraftSpec struct {
	// title is what the draft is called, and nil for the draft that has never
	// been given one. A draft keeps its own title apart from the page's, and
	// the listings fall back to the page's title and then to an untitled label,
	// so a draft without one is the only way to see that fallback reached.
	//
	// [Ja] title は下書きの呼び名で、一度もタイトルを付けられていない下書きでは
	// nil になる。下書きはページとは別に自身のタイトルを持ち、一覧はページの
	// タイトル、次いで「無題」の表示へフォールバックする。タイトルの無い下書きは、
	// そのフォールバックの末端を確認できる唯一の手段になる。
	title *string
	// intro opens the body. What follows it is written by the saves.
	//
	// [Ja] intro は本文の書き出し。その後ろは保存が書き足す。
	intro string
	// longHistory asks for the draft to be saved as many times as the amounts
	// ask for, instead of the couple of times the others are.
	//
	// [Ja] longHistory は、他の下書きのように数回ではなく、件数の設定が求める
	// 回数だけ保存されることを求める。
	longHistory bool
}

// newPageDraftSpecs are the drafts on unpublished pages. Between them they
// cover what a draft can be before its page has ever been published: one that
// is simply unfinished, one that has been saved more times than its edit
// history shows at once, and one that has never been given a title.
//
// They are a fixed list rather than a count in the amounts because each one is
// there for a state of its own, the way the sandbox pages are.
//
// [Ja] newPageDraftSpecs は未公開のページに付く下書き。全体で、ページが一度も
// 公開されていない段階の下書きが取りうる状態を網羅する。単に書きかけのもの、
// 編集履歴が一度に見せる件数より多く保存されたもの、そして一度もタイトルを
// 付けられていないもの。
//
// 件数の設定ではなく固定の一覧にしているのは、それぞれが固有の状態のために存在
// するためで、表示崩れ確認用ページと同じ扱いになる。
func newPageDraftSpecs() []newPageDraftSpec {
	unpublished := unpublishedDraftTitle
	longHistory := longHistoryDraftTitle

	return []newPageDraftSpec{
		{
			title: &unpublished,
			intro: `この下書きは、一度も公開されていないページに対して書かれています。ページ自体は存在しますが (ページ作成ボタンが作りました)、タイトルも本文も持たず、どの一覧にも出てきません。そこへ辿り着けるのは、この下書きだけです。`,
		},
		{
			title:       &longHistory,
			intro:       `この下書きは、編集履歴が一度に見せる件数より多く保存されています。履歴は新しい保存から順に並び、上限を超えた古い保存は一覧に出てきません。各エントリの番号は最も古い保存を 1 番とする通し番号のため、一覧の末尾が 1 番で終わっていないことが、ここに出ていない保存があることを示します。`,
			longHistory: true,
		},
		{
			intro: `この下書きには一度もタイトルが付けられておらず、対象のページにも付いていません。この下書きを見せる 2 つの一覧は、どちらも「無題」の表示にフォールバックします。この下書きは、その状態を見せるために存在します。`,
		},
	}
}

// generateDraftPages creates the drafts the home screen, the draft listing and
// the editor's two side columns are read from.
//
// A draft is what an edit looks like before it is published, and it is private
// to the member who wrote it. Both accounts therefore get drafts of their own:
// a listing that is only ever populated for one account cannot be checked from
// the other.
//
// [Ja] generateDraftPages は、ホーム画面・下書き一覧画面・編集画面の 2 つの
// サイドカラムが読む下書きを作成する。
//
// 下書きは公開前の編集内容であり、書いたメンバー本人だけのものである。そのため
// 両方のアカウントに下書きを持たせる。片方のアカウントでしか埋まらない一覧は、
// もう片方から確認できないため。
func generateDraftPages(
	ctx context.Context,
	dbtx query.DBTX,
	out io.Writer,
	amt amounts,
	spaces *seededSpaces,
	topics *seededTopics,
	stamps *draftStamps,
) error {
	specs := newPageDraftSpecs()

	onPublishedPages := amt.ownerDraftPages - len(specs)
	if onPublishedPages < 0 {
		return fmt.Errorf(
			"ownerの下書き %d 件は、未公開のページに付ける %d 件を下回っている",
			amt.ownerDraftPages, len(specs),
		)
	}

	bar := newProgress(out, "下書きページ", amt.ownerDraftPages+amt.collaboratorDraftPages)
	defer bar.finish()

	owner, err := spaces.wiki.requireMember(roleOwner)
	if err != nil {
		return err
	}

	collaborator, err := spaces.wiki.requireMember(roleCollaborator)
	if err != nil {
		return err
	}

	// The drafts are spread over the topics both accounts can read. The draft
	// listing groups by space and topic, so a single topic would leave that
	// screen with one group and nothing to tell the grouping by.
	//
	// [Ja] 下書きは両アカウントが読めるトピックへ分散させる。下書き一覧画面は
	// スペースとトピックでグループ分けするため、1 つのトピックに寄せるとその画面が
	// 1 グループになり、グループ分けを見分けるものが無くなる。
	draftTopics := []*seededTopic{topics.handbook, topics.notes, topics.privateNotes}

	targets, err := collectDraftTargets(ctx, dbtx, draftTopics, onPublishedPages+amt.collaboratorDraftPages)
	if err != nil {
		return err
	}

	writer := newDraftWriter(dbtx, spaces.wiki)

	// The drafts on unpublished pages are created first, which makes them the
	// most recently modified ones. The home screen keeps only the newest few,
	// and these are the drafts worth meeting there.
	//
	// [Ja] 未公開のページに付く下書きを先に作成し、最終更新が最も新しい状態に
	// する。ホーム画面は新しいものを数件しか残さず、そこで出会う価値があるのは
	// これらの下書きであるため。
	for _, spec := range specs {
		revisions := ordinaryDraftRevisions
		if spec.longHistory {
			revisions = amt.draftRevisions
		}

		if err := writer.createNewPageDraft(ctx, newPageDraftInput{
			topic:      topics.notes,
			member:     owner,
			spec:       spec,
			revisions:  revisions,
			modifiedAt: stamps.next(),
		}); err != nil {
			return err
		}
		bar.advance()
	}

	// The two accounts draft against pages of their own. Sharing a page would
	// be a state the application allows, but it would also put the same title
	// in both listings and make them hard to tell apart while browsing.
	//
	// [Ja] 2 つのアカウントはそれぞれ別のページに対して下書きを書く。同じページを
	// 共有する状態はアプリケーションが許すものではあるが、両方の一覧に同じ
	// タイトルが並ぶことになり、閲覧しながら見分けるのが難しくなる。
	for _, group := range []struct {
		member  *seededSpaceMember
		targets []draftTarget
	}{
		{member: owner, targets: targets[:onPublishedPages]},
		{member: collaborator, targets: targets[onPublishedPages:]},
	} {
		for _, target := range group.targets {
			if err := writer.createPublishedPageDraft(ctx, target, group.member, stamps.next()); err != nil {
				return err
			}
			bar.advance()
		}
	}

	return nil
}

// draftTarget is a published page a draft is written against, together with the
// topic it sits in. The topic travels with the page because a draft carries a
// topic of its own, and because rendering the draft's body needs the name of
// the topic the body is read from.
//
// [Ja] draftTarget は、下書きが対象とする公開済みのページと、それが属する
// トピック。トピックをページと一緒に持つのは、下書き自身がトピックを持つことと、
// 下書きの本文のレンダリングに、本文が読まれるトピックの名前が要ることによる。
type draftTarget struct {
	topic *seededTopic
	page  *seededPage
}

// collectDraftTargets picks the published pages the drafts are written against,
// taking them from the topics in turn so that every topic ends up with drafts
// in it.
//
// The pages are taken from what earlier generators published rather than
// created here. The space-wide page listing is sized to leave a partial last
// page, and pages added for the sake of drafts would move it off that size for
// a reason that has nothing to do with what a draft is.
//
// [Ja] collectDraftTargets は、下書きが対象とする公開済みのページを選ぶ。
// トピックから順番に取ることで、どのトピックにも下書きが行き渡るようにする。
//
// ページはここで作成せず、先行する生成器が公開したものから取る。スペース全体の
// ページ一覧は最終ページが端数になる件数に調整されており、下書きのために
// ページを足すと、下書きとは無関係な理由でその件数からずれてしまうため。
func collectDraftTargets(
	ctx context.Context,
	dbtx query.DBTX,
	topicList []*seededTopic,
	count int,
) ([]draftTarget, error) {
	if count == 0 {
		return nil, nil
	}

	perTopic := (count + len(topicList) - 1) / len(topicList)

	byTopic := make([][]draftTarget, len(topicList))
	for i, topic := range topicList {
		pages, err := listDraftTargetPages(ctx, dbtx, topic, perTopic)
		if err != nil {
			return nil, err
		}
		if len(pages) < perTopic {
			return nil, fmt.Errorf(
				"トピック %s の下書き対象にできるページが %d 件しかなく、必要な %d 件に足りない",
				topic.name, len(pages), perTopic,
			)
		}
		byTopic[i] = pages
	}

	targets := make([]draftTarget, 0, count)
	for i := 0; len(targets) < count; i++ {
		targets = append(targets, byTopic[i%len(topicList)][i/len(topicList)])
	}

	return targets, nil
}

// listDraftTargetPages returns the pages of one topic that a draft may be
// written against, oldest first.
//
// Pinned and trashed pages are left out. A trashed page is reached from the
// trash alone, so a draft of one would sit in the draft listings pointing at a
// page the listings themselves no longer show, and a pinned page is displayed
// apart from the listing the other targets are met in.
//
// [Ja] listDraftTargetPages は、あるトピックのうち下書きの対象にできるページを
// 古い順に返す。
//
// ピン留めされたページとゴミ箱のページは除く。ゴミ箱のページはゴミ箱からしか
// 辿れないため、その下書きは、一覧自身がもう見せていないページを指したまま下書きの
// 一覧に並ぶことになる。ピン留めされたページは、他の対象と出会う一覧とは別の場所に
// 表示される。
func listDraftTargetPages(
	ctx context.Context,
	dbtx query.DBTX,
	topic *seededTopic,
	limit int,
) ([]draftTarget, error) {
	rows, err := dbtx.QueryContext(
		ctx,
		`SELECT id, number, title
         FROM pages
         WHERE space_id = $1
           AND topic_id = $2
           AND published_at IS NOT NULL
           AND pinned_at IS NULL
           AND trashed_at IS NULL
           AND discarded_at IS NULL
         ORDER BY number
         LIMIT $3`,
		string(topic.spaceID), string(topic.id), limit,
	)
	if err != nil {
		return nil, fmt.Errorf("トピック %s の公開済みページの取得に失敗: %w", topic.name, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	targets := make([]draftTarget, 0, limit)
	for rows.Next() {
		var (
			id     string
			number int32
			title  *string
		)
		if err := rows.Scan(&id, &number, &title); err != nil {
			return nil, fmt.Errorf("トピック %s の公開済みページの読み取りに失敗: %w", topic.name, err)
		}

		page := &seededPage{id: model.PageID(id), number: model.PageNumber(number)}
		if title != nil {
			page.title = *title
		}
		targets = append(targets, draftTarget{topic: topic, page: page})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("トピック %s の公開済みページの読み取りに失敗: %w", topic.name, err)
	}

	return targets, nil
}

// draftWriter creates drafts and their revisions in one space. It holds a
// pageWriter of its own because a draft of a page that was never published
// needs that page created first, and because a draft body is rendered through
// the same path a page body is.
//
// [Ja] draftWriter は 1 つのスペースに下書きとそのリビジョンを作成する。自前の
// pageWriter を持つのは、一度も公開されていないページの下書きには、まずその
// ページの作成が必要であることと、下書きの本文がページの本文と同じ経路で
// レンダリングされることによる。
type draftWriter struct {
	space                 *seededSpace
	pages                 *pageWriter
	draftPageRepo         *repository.DraftPageRepository
	draftPageRevisionRepo *repository.DraftPageRevisionRepository
}

// newDraftWriter returns a writer that creates drafts in space.
//
// [Ja] newDraftWriter は space に下書きを作成する writer を返す。
func newDraftWriter(dbtx query.DBTX, space *seededSpace) *draftWriter {
	queries := query.New(dbtx)

	return &draftWriter{
		space:                 space,
		pages:                 newPageWriter(dbtx, space),
		draftPageRepo:         repository.NewDraftPageRepository(queries),
		draftPageRevisionRepo: repository.NewDraftPageRevisionRepository(queries),
	}
}

// newPageDraftInput describes one draft on a page that has never been
// published.
//
// [Ja] newPageDraftInput は、一度も公開されていないページに付く下書き 1 件の内容。
type newPageDraftInput struct {
	topic      *seededTopic
	member     *seededSpaceMember
	spec       newPageDraftSpec
	revisions  int
	modifiedAt time.Time
}

// createNewPageDraft creates the page the new page button leaves behind and the
// draft written against it.
//
// [Ja] createNewPageDraft は、ページ作成ボタンが残すページと、それに対して
// 書かれた下書きを作成する。
func (w *draftWriter) createNewPageDraft(ctx context.Context, input newPageDraftInput) error {
	page, err := w.pages.createBlankPage(ctx, input.topic, input.member)
	if err != nil {
		return err
	}

	return w.createDraft(ctx, createDraftInput{
		topic:      input.topic,
		member:     input.member,
		page:       page,
		title:      input.spec.title,
		intro:      input.spec.intro,
		revisions:  input.revisions,
		modifiedAt: input.modifiedAt,
	})
}

// createPublishedPageDraft creates a draft of a page that is already published.
// The draft keeps the page's title, which is what an edit that changes only the
// body leaves behind.
//
// [Ja] createPublishedPageDraft は、既に公開されているページの下書きを作成する。
// 下書きはページのタイトルをそのまま持つ。これは本文だけを変える編集が残す形に
// あたる。
func (w *draftWriter) createPublishedPageDraft(
	ctx context.Context,
	target draftTarget,
	member *seededSpaceMember,
	modifiedAt time.Time,
) error {
	title := target.page.title

	return w.createDraft(ctx, createDraftInput{
		topic:      target.topic,
		member:     member,
		page:       target.page,
		title:      &title,
		intro:      publishedPageDraftIntro(target.page.title),
		revisions:  ordinaryDraftRevisions,
		modifiedAt: modifiedAt,
	})
}

// createDraftInput describes one draft to create.
//
// [Ja] createDraftInput は作成する下書き 1 件の内容。
type createDraftInput struct {
	topic  *seededTopic
	member *seededSpaceMember
	page   *seededPage
	title  *string
	intro  string
	// revisions is how many times the draft has been saved. The draft keeps
	// what the last save wrote, and the edit history keeps all of them.
	//
	// [Ja] revisions は下書きが保存された回数。下書きは最後の保存が書いたものを
	// 保持し、編集履歴はそのすべてを保持する。
	revisions  int
	modifiedAt time.Time
}

// label names the draft in an error message. A draft that has never been given
// a title has nothing else to be named by.
//
// [Ja] label はエラーメッセージ内で下書きを名指しする。一度もタイトルを
// 付けられていない下書きには、他に名指しするものが無いため。
func (input createDraftInput) label() string {
	if input.title != nil && *input.title != "" {
		return *input.title
	}

	return "タイトル未設定の下書き"
}

// createDraft creates one draft together with the revision each of its saves
// left behind, which is the set of rows saving from the editor produces.
//
// Both go through their repositories: saving a draft is already handled by the
// Go side, so the Create the screen calls is the Create the seed calls.
//
// [Ja] createDraft は下書き 1 件と、その各保存が残したリビジョンを作成する。
// これは編集画面から保存したときに残る行の一式にあたる。
//
// どちらも Repository を経由する。下書きの保存は既に Go 側が担当しているため、
// 画面が呼ぶ Create をそのままシードも呼べる。
func (w *draftWriter) createDraft(ctx context.Context, input createDraftInput) error {
	if err := w.pages.ensureTopicInSpace(input.topic); err != nil {
		return err
	}

	var title string
	if input.title != nil {
		title = *input.title
	}

	// Each save appends a line, so the history reads as a body being written
	// over time rather than as the same text stored again and again, and the
	// diff of any revision shows the one line that save added.
	//
	// [Ja] 保存のたびに 1 行を書き足す。これにより履歴が、同じ本文の保存の
	// 繰り返しではなく、時間をかけて書かれていく本文として読め、どのリビジョンの
	// 差分もその保存が足した 1 行を見せるようになる。
	bodies := make([]string, input.revisions)
	rendered := make([]string, input.revisions)
	var linkedPageIDs []model.PageID
	for i := range bodies {
		bodies[i] = draftRevisionBody(input.intro, i+1)

		html, linked, err := w.pages.render(ctx, createPageInput{
			topic:  input.topic,
			author: input.member,
			title:  input.label(),
			body:   bodies[i],
		})
		if err != nil {
			return err
		}
		rendered[i] = html
		linkedPageIDs = linked
	}

	last := input.revisions - 1

	draftPage, err := w.draftPageRepo.Create(ctx, repository.CreateDraftPageInput{
		SpaceID:       w.space.id,
		PageID:        input.page.id,
		SpaceMemberID: input.member.id,
		TopicID:       input.topic.id,
		Title:         input.title,
		Body:          bodies[last],
		BodyHTML:      rendered[last],
		LinkedPageIDs: linkedPageIDs,
		ModifiedAt:    input.modifiedAt,
	})
	if err != nil {
		return fmt.Errorf("下書き %s の作成に失敗: %w", input.label(), err)
	}

	for i := range bodies {
		if _, err := w.draftPageRevisionRepo.Create(ctx, repository.CreateDraftPageRevisionInput{
			DraftPageID:   draftPage.ID,
			SpaceID:       w.space.id,
			SpaceMemberID: input.member.id,
			Title:         title,
			Body:          bodies[i],
			BodyHTML:      rendered[i],
		}); err != nil {
			return fmt.Errorf("下書き %s のリビジョンの作成に失敗: %w", input.label(), err)
		}
	}

	return nil
}

// draftRevisionBody builds the body the draft held after the given save.
//
// [Ja] draftRevisionBody は、指定の保存を終えた時点で下書きが持っていた本文を
// 組み立てる。
func draftRevisionBody(intro string, version int) string {
	var b strings.Builder

	b.WriteString(intro)
	b.WriteString("\n\n")

	for v := 1; v <= version; v++ {
		fmt.Fprintf(&b, "- %d 回目の保存がこの行を足しました。\n", v)
	}

	return b.String()
}

// publishedPageDraftIntro opens the body of a draft written against a page that
// is already published. The title is written into Japanese prose, so it is
// spaced by what it begins and ends with.
//
// [Ja] publishedPageDraftIntro は、既に公開されているページに対して書かれた
// 下書きの本文の書き出しを組み立てる。タイトルは日本語の文の中に書き込まれるため、
// その始まりと終わりの文字によって空白を補う。
func publishedPageDraftIntro(pageTitle string) string {
	return fmt.Sprintf(`これは%sの未公開の編集です。下書きは、公開されるまでページ本体とは別に保持されます。そのためページ側は最後に公開された内容を見せたままで、編集画面が代わりに開くのがこの下書きになります。`, spacedInJapanese(pageTitle))
}

// spacedInJapanese returns s with a space added on whichever side meets the
// surrounding Japanese text with an ASCII character.
//
// Whether a space belongs is decided by the two characters that meet, not by
// the string as a whole: the pages a draft is written against are titled
// `ハンドブック 001`, `リンクハブ` and `Markdown 記法`, which between them open
// and close with both kinds of character.
//
// [Ja] spacedInJapanese は、前後を日本語の文に挟まれた s のうち、ASCII 文字で接する側に
// 空白を足して返す。
//
// 空白の要否は接する 2 つの文字で決まり、文字列全体では決まらない。下書きが対象と
// するページのタイトルは `ハンドブック 001`・`リンクハブ`・`Markdown 記法` であり、
// 全体で見れば半角と全角のどちらでも始まり、どちらでも終わるため。
func spacedInJapanese(s string) string {
	if s == "" {
		return s
	}

	first, _ := utf8.DecodeRuneInString(s)
	last, _ := utf8.DecodeLastRuneInString(s)

	if first < utf8.RuneSelf {
		s = " " + s
	}
	if last < utf8.RuneSelf {
		s += " "
	}

	return s
}

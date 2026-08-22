package seed

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// Titles of the suggestions written to show a state the rest do not reach.
// They are named rather than numbered because each one is there for a state of
// its own, and because the listing shows a suggestion under its own title.
//
// None of them names the topic it sits in. The listing that shows them belongs
// to a topic, so naming it would repeat what the screen already says, and a
// title that names one would be left pointing at a name that no longer exists
// once the topic is renamed.
//
// [Ja] 他の編集提案では届かない状態を見せるために書いた編集提案のタイトル。番号では
// なく名前を付けているのは、それぞれが固有の状態のために存在することと、一覧が
// 編集提案自身のタイトルで表示することによる。
//
// いずれのタイトルも、自身が属するトピックを名指ししない。これらを見せる一覧は
// トピックに属しているため、名指しは画面が既に言っていることの繰り返しになり、
// トピックの名前が変わったときには、存在しない名前を指したタイトルが残ることになる。
const (
	multiPageSuggestionTitle   = "3 ページをまとめて書き直す"
	discussionSuggestionTitle  = "どのページも導入文を繰り返すべきか?"
	editStartedSuggestionTitle = "ページの書き出しを言い換える"
)

// ordinarySuggestionTitleFormat names a suggestion after the page it proposes
// to change. The listing shows the title and nothing of the pages behind it, so
// naming them apart is what lets one row be told from the next.
//
// [Ja] ordinarySuggestionTitleFormat は、編集提案を、それが変更を提案するページに
// ちなんで命名する。一覧はタイトルだけを見せ、その先のページについては何も見せない
// ため、行と行を見分けられるのは名前が違うことによる。
const ordinarySuggestionTitleFormat = "%s を修正する"

// renamedPageTitleSuffix is appended to the title of the one page a suggestion
// proposes to rename. The changed pages screen shows a title change apart from
// the body diff, and that part of the screen is only reached by a suggestion
// that carries one.
//
// [Ja] renamedPageTitleSuffix は、編集提案がタイトルの変更を提案する唯一のページの
// タイトルに付け足される。変更差分画面はタイトルの変更を本文の差分とは別に表示し、
// その部分はタイトルの変更を含む編集提案からしか到達できない。
const renamedPageTitleSuffix = " (改題)"

// showcaseSuggestionSpec describes one suggestion written to show a state the
// ordinary ones do not: several changed pages at once, a discussion longer than
// the suggestion itself, or an edit that is still under way.
//
// They are a fixed list rather than a count in the amounts because each one is
// there for a state of its own, the way the drafts on unpublished pages are.
//
// [Ja] showcaseSuggestionSpec は、通常の編集提案では見せられない状態を見せるために
// 書いた編集提案 1 件の内容。一度に複数のページを変更するもの、編集提案本体より長い
// 議論が付いたもの、そして編集がまだ進行中のもの。
//
// 件数の設定ではなく固定の一覧にしているのは、それぞれが固有の状態のために存在する
// ためで、未公開のページに付く下書きと同じ扱いになる。
type showcaseSuggestionSpec struct {
	title string
	body  string
	// changedPages is how many of the topic's pages the suggestion proposes to
	// change. The changed pages screen holds one diff per page, so more than
	// one is what shows how those diffs sit next to each other.
	//
	// [Ja] changedPages は、その編集提案がトピックのページを何件変更しようとして
	// いるか。変更差分画面は 1 ページにつき 1 つの差分を持つため、複数あることで
	// 差分同士がどう並ぶのかが見える。
	changedPages int
	// renamesLastPage asks for the last of the changed pages to be given a new
	// title, which is the only way the title part of a diff is reached.
	//
	// [Ja] renamesLastPage は、変更対象の最後のページに新しいタイトルを付けることを
	// 求める。差分のタイトル部分に到達する唯一の手段であるため。
	renamesLastPage bool
	// manyComments asks for the discussion to run as long as the amounts ask
	// for, instead of leaving the suggestion without one.
	//
	// [Ja] manyComments は、議論を件数の設定が求める長さまで伸ばすことを求める。
	// 議論が付かないままにするのではなく。
	manyComments bool
	// editStarted asks for a draft linked to the suggestion page, which is what
	// starting to edit a suggested page leaves behind. The page detail screen
	// reports the suggestion when the member who started it opens the page, and
	// no other suggestion puts the screen in that state.
	//
	// [Ja] editStarted は、編集提案ページに紐づく下書きを求める。これは提案された
	// ページの編集を始めたときに残るものにあたる。ページ詳細画面は、編集を始めた
	// メンバーがそのページを開いたときに編集提案を知らせる。他のどの編集提案も画面を
	// その状態にしないため。
	editStarted bool
}

// showcaseSuggestionSpecs are the suggestions that show a state of their own.
//
// [Ja] showcaseSuggestionSpecs は、それぞれ固有の状態を見せる編集提案。
func showcaseSuggestionSpecs() []showcaseSuggestionSpec {
	return []showcaseSuggestionSpec{
		{
			title:           multiPageSuggestionTitle,
			body:            `この編集提案は 3 つのページを一度に変更し、その最後のページの名前も変更します。変更差分タブは 1 ページにつき 1 つの差分を持つため、差分同士がどう並ぶのかと、名前を変更したページが本文だけの編集とどう見分けられるのかが、どちらも見えます。`,
			changedPages:    3,
			renamesLastPage: true,
		},
		{
			title:        discussionSuggestionTitle,
			body:         `この編集提案は、議論を持たせるために存在します。下に並ぶコメントは 2 つのアカウントが交互に書いたもので、始まりになった編集提案よりも長くなった会話がどう見えるのかが分かります。`,
			changedPages: 1,
			manyComments: true,
		},
		{
			title:        editStartedSuggestionTitle,
			body:         `この編集提案が変更を提案しているページは、いま誰かが編集している最中です。その編集画面が持つ下書きは下の提案ページに紐づいており、ページ詳細画面は、そうした編集が進んでいる間これを知らせます。`,
			changedPages: 1,
			editStarted:  true,
		},
	}
}

// generateSuggestions creates the suggestions that the topic's suggestion
// listing, the discussion under a suggestion and the changed pages screen are
// read from.
//
// A suggestion is an edit proposed rather than published: it holds the pages it
// would change, the discussion about it, and the mark of how it ended once it
// is settled. All of them are created in one topic, because the listing that
// shows suggestions belongs to a topic and spreading them would leave every one
// of those listings short.
//
// topics.handbook is that topic. Its pages are the ones written to be counted
// by a listing, and an applied suggestion rewrites the page it names: the
// topics that hold pages written to be read would lose what they were written
// for.
//
// [Ja] generateSuggestions は、トピックの編集提案一覧・編集提案の議論・変更差分画面が
// 読む編集提案を作成する。
//
// 編集提案は、公開ではなく提案として行われる編集であり、変更対象のページ・それに
// ついての議論・決着後にはどう終わったかの印を持つ。すべてを 1 つのトピックに作るのは、
// 編集提案の一覧がトピックに属しており、分散させるとどの一覧も短いままになるため。
//
// そのトピックは「ハンドブック」になる。あそこのページは一覧に数えられるために書かれた
// ものであり、反映済みの編集提案は名指ししたページを書き換える。読ませるために書かれた
// ページを持つトピックでは、そのページが書かれた目的が失われてしまう。
func generateSuggestions(
	ctx context.Context,
	dbtx query.DBTX,
	out io.Writer,
	amt amounts,
	spaces *seededSpaces,
	topics *seededTopics,
	stamps *draftStamps,
) error {
	specs := showcaseSuggestionSpecs()

	ordinaryOpen := amt.openSuggestions - len(specs)
	if ordinaryOpen < 0 {
		return fmt.Errorf(
			"オープンな編集提案 %d 件は、状態を見せるために固定で作る %d 件を下回っている",
			amt.openSuggestions, len(specs),
		)
	}

	bar := newProgress(out, "編集提案", amt.openSuggestions+amt.appliedSuggestions+amt.closedSuggestions)
	defer bar.finish()

	pagesNeeded := ordinaryOpen + amt.appliedSuggestions + amt.closedSuggestions
	for _, spec := range specs {
		pagesNeeded += spec.changedPages
	}

	targets, err := collectSuggestionTargets(ctx, dbtx, topics.handbook, pagesNeeded)
	if err != nil {
		return err
	}

	// take hands out the next pages to be proposed against. Each suggestion
	// gets pages of its own: two suggestions changing the same page is a state
	// the application allows, but it would also put the same page under two
	// diffs and make them hard to tell apart while browsing.
	//
	// [Ja] take は、次に変更対象とするページを渡す。各編集提案はそれぞれ別のページを
	// 受け取る。同じページを 2 つの編集提案が変更する状態はアプリケーションが許すもの
	// ではあるが、同じページが 2 つの差分に並ぶことになり、閲覧しながら見分けるのが
	// 難しくなる。
	take := func(count int) []suggestionTarget {
		taken := targets[:count]
		targets = targets[count:]

		return taken
	}

	owner, err := spaces.wiki.requireMember(roleOwner)
	if err != nil {
		return err
	}

	writer := newSuggestionWriter(dbtx, spaces.wiki, owner, stamps)

	// position counts the suggestions as they are created, and decides which
	// account each one is attributed to.
	//
	// [Ja] position は作成順に編集提案を数え、それぞれをどのアカウントのものとして
	// 記録するかを決める。
	position := 0

	// The ordinary suggestions come first, and the ones written to show a state
	// of their own last. The listing orders suggestions by when they were
	// created, newest first, so the last ones created are the ones met at the
	// top of it.
	//
	// [Ja] 通常の編集提案を先に、固有の状態を見せるために書いたものを最後に作成する。
	// 一覧は作成時刻の新しい順に並べるため、最後に作成したものが一覧の先頭で
	// 出会うものになる。
	for _, group := range []struct {
		count  int
		status model.SuggestionStatus
	}{
		{count: ordinaryOpen, status: model.SuggestionStatusOpen},
		{count: amt.appliedSuggestions, status: model.SuggestionStatusApplied},
		{count: amt.closedSuggestions, status: model.SuggestionStatusClosed},
	} {
		for i := 0; i < group.count; i++ {
			position++
			target := take(1)[0]

			creator, err := suggestionCreator(spaces.wiki, position)
			if err != nil {
				return err
			}

			if err := writer.createSuggestion(ctx, createSuggestionInput{
				topic:   topics.handbook,
				creator: creator,
				title:   fmt.Sprintf(ordinarySuggestionTitleFormat, target.page.title),
				body:    ordinarySuggestionBody(target.page.title),
				targets: []suggestionTarget{target},
				status:  group.status,
			}); err != nil {
				return err
			}
			bar.advance()
		}
	}

	for _, spec := range specs {
		position++

		comments := 0
		if spec.manyComments {
			comments = amt.suggestionComments
		}

		creator, err := suggestionCreator(spaces.wiki, position)
		if err != nil {
			return err
		}

		if err := writer.createSuggestion(ctx, createSuggestionInput{
			topic:           topics.handbook,
			creator:         creator,
			title:           spec.title,
			body:            spec.body,
			targets:         take(spec.changedPages),
			status:          model.SuggestionStatusOpen,
			renamesLastPage: spec.renamesLastPage,
			comments:        comments,
			editStarted:     spec.editStarted,
		}); err != nil {
			return err
		}
		bar.advance()
	}

	return nil
}

// suggestionCreator picks the account a suggestion of the given position is
// attributed to, handing the suggestions round the roles.
//
// What a member may do with a suggestion depends on whether they opened it:
// the creator may close their own, while applying it takes the administrator.
// Suggestions from one account alone would leave one side of that line without
// anything to look at.
//
// [Ja] suggestionCreator は、その位置の編集提案を誰のものとして記録するかを選ぶ。
// 編集提案は役割へ順に回される。
//
// メンバーが編集提案に対して何をできるかは、それを自分が開いたかどうかで変わる。
// 作成者は自分のものをクローズでき、反映するには管理者である必要がある。1 つの
// アカウントだけが編集提案を持つと、この線のどちらか側に見るものが無くなる。
func suggestionCreator(space *seededSpace, position int) (*seededSpaceMember, error) {
	return space.memberInTurn(contentAuthorRoles, position)
}

// suggestionCommentAuthor picks the account that writes the comment at the
// given position in a thread, handing the remarks round the roles.
//
// A thread written by one account alone reads as a member talking to
// themselves. The discussion is looked at to see how remarks by different
// accounts sit against each other, so who each one comes from is what there is
// to see.
//
// This is kept apart from suggestionCreator, which hands its work round for a
// reason of its own: what a member may do with a suggestion depends on whether
// they opened it. Sharing one function would leave whichever call site is read
// second explained by the other one's reason.
//
// [Ja] suggestionCommentAuthor は、スレッドの指定位置のコメントを書くアカウントを
// 選ぶ。発言は役割へ順に回される。
//
// 1 つのアカウントだけが書いたスレッドは、メンバーの独り言として読まれる。議論は
// 別々のアカウントの発言が互いにどう並ぶかを見るために眺められるため、各発言が
// 誰のものかが見るべきものになる。
//
// suggestionCreator と分けているのは、あちらが別の理由で順に回しているため。
// メンバーが編集提案に対して何をできるかは、それを自分が開いたかどうかで変わる。
// 1 つの関数を共有すると、後から読んだほうの呼び出し側が、もう一方の理由で説明されて
// しまう。
func suggestionCommentAuthor(space *seededSpace, position int) (*seededSpaceMember, error) {
	return space.memberInTurn(contentAuthorRoles, position)
}

// suggestionTarget is a published page a suggestion proposes to change,
// together with the body that page holds now.
//
// The body travels with the page because the proposed body is built from it.
// The diff is taken between the two, so a body written without looking at what
// is on the page would show up as one deletion and one addition with no
// unchanged line between them.
//
// [Ja] suggestionTarget は、編集提案が変更を提案する公開済みのページと、その
// ページが現在保持している本文。
//
// 本文をページと一緒に持つのは、提案する本文がそこから組み立てられるため。差分は
// 両者の間で取られるため、ページにあるものを見ずに書いた本文は、間に変更のない行を
// 挟まない 1 つの削除と 1 つの追加として表示されてしまう。
type suggestionTarget struct {
	page *seededPage
	body string
}

// collectSuggestionTargets picks the published pages the suggestions are
// written against.
//
// The pages are taken from what earlier generators published rather than
// created here, for the same reason the drafts take theirs: the space-wide page
// listing is sized to leave a partial last page, and pages added for the sake
// of suggestions would move it off that size for a reason that has nothing to
// do with what a suggestion is.
//
// Pages a draft is already written against are left out. Such a page is being
// edited by its own member, and proposing a change to it as well would leave
// the editor asking which of the two to keep before either can be looked at.
//
// [Ja] collectSuggestionTargets は、編集提案が対象とする公開済みのページを選ぶ。
//
// ページはここで作成せず、先行する生成器が公開したものから取る。下書きが対象を
// そうしているのと同じ理由による。スペース全体のページ一覧は最終ページが端数になる
// 件数に調整されており、編集提案のためにページを足すと、編集提案とは無関係な理由で
// その件数からずれてしまう。
//
// 既に下書きが書かれているページは除く。そのページはそのメンバー自身が編集している
// ところであり、そこへさらに変更を提案すると、どちらを残すのかを編集画面が尋ねてから
// でないとどちらも見られなくなる。
func collectSuggestionTargets(
	ctx context.Context,
	dbtx query.DBTX,
	topic *seededTopic,
	count int,
) ([]suggestionTarget, error) {
	if count == 0 {
		return nil, nil
	}

	rows, err := dbtx.QueryContext(
		ctx,
		`SELECT p.id, p.number, p.title, p.body
         FROM pages p
         WHERE p.space_id = $1
           AND p.topic_id = $2
           AND p.published_at IS NOT NULL
           AND p.title IS NOT NULL
           AND p.pinned_at IS NULL
           AND p.trashed_at IS NULL
           AND p.discarded_at IS NULL
           AND NOT EXISTS (
             SELECT 1 FROM draft_pages d
             WHERE d.space_id = p.space_id AND d.page_id = p.id
           )
         ORDER BY p.number
         LIMIT $3`,
		string(topic.spaceID), string(topic.id), count,
	)
	if err != nil {
		return nil, fmt.Errorf("トピック %s の公開済みページの取得に失敗: %w", topic.name, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	targets := make([]suggestionTarget, 0, count)
	for rows.Next() {
		var (
			id     string
			number int32
			title  string
			body   string
		)
		if err := rows.Scan(&id, &number, &title, &body); err != nil {
			return nil, fmt.Errorf("トピック %s の公開済みページの読み取りに失敗: %w", topic.name, err)
		}

		targets = append(targets, suggestionTarget{
			page: &seededPage{id: model.PageID(id), number: model.PageNumber(number), title: title},
			body: body,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("トピック %s の公開済みページの読み取りに失敗: %w", topic.name, err)
	}

	if len(targets) < count {
		return nil, fmt.Errorf(
			"トピック %s の編集提案の対象にできるページが %d 件しかなく、必要な %d 件に足りない",
			topic.name, len(targets), count,
		)
	}

	return targets, nil
}

// suggestionWriter creates suggestions in one space. It holds a pageWriter of
// its own because a proposed body is rendered through the same path a page body
// is, and because applying a suggestion writes onto the pages themselves.
//
// [Ja] suggestionWriter は 1 つのスペースに編集提案を作成する。自前の pageWriter を
// 持つのは、提案された本文がページの本文と同じ経路でレンダリングされることと、
// 編集提案の反映がページ自体へ書き込むことによる。
type suggestionWriter struct {
	space *seededSpace
	// owner is the member that applies a suggestion and that a started edit
	// leaves a draft for. Both take the administrator, so the role is resolved
	// once when the writer is built rather than looked up at each write.
	//
	// [Ja] owner は、編集提案を反映するメンバーであり、編集を始めたときの下書きが
	// 残る先でもある。どちらも管理者であることを要するため、書き込みのたびに引かず、
	// writer を組み立てるときに 1 度だけ解決する。
	owner                      *seededSpaceMember
	pages                      *pageWriter
	suggestionRepo             *repository.SuggestionRepository
	suggestionPageRepo         *repository.SuggestionPageRepository
	suggestionPageRevisionRepo *repository.SuggestionPageRevisionRepository
	suggestionCommentRepo      *repository.SuggestionCommentRepository
	draftPageRepo              *repository.DraftPageRepository
	topicMemberRepo            *repository.TopicMemberRepository
	// draftStamps stamps the draft a started edit leaves behind. It is the
	// counter the whole run shares, so that draft stands behind the drafts the
	// draft phase wrote instead of ahead of them.
	//
	// [Ja] draftStamps は、編集を始めたときに残る下書きを打刻する。実行全体で共有
	// されるカウンターであり、この下書きが下書きフェーズの書いた下書きより前ではなく
	// 後ろに並ぶようにするためのもの。
	draftStamps *draftStamps
}

// newSuggestionWriter returns a writer that creates suggestions in space,
// applying them and starting edits as owner.
//
// [Ja] newSuggestionWriter は space に編集提案を作成する writer を返す。反映と
// 編集の開始は owner として行う。
func newSuggestionWriter(
	dbtx query.DBTX,
	space *seededSpace,
	owner *seededSpaceMember,
	stamps *draftStamps,
) *suggestionWriter {
	queries := query.New(dbtx)

	return &suggestionWriter{
		space:                      space,
		owner:                      owner,
		pages:                      newPageWriter(dbtx, space),
		suggestionRepo:             repository.NewSuggestionRepository(queries),
		suggestionPageRepo:         repository.NewSuggestionPageRepository(queries),
		suggestionPageRevisionRepo: repository.NewSuggestionPageRevisionRepository(queries),
		suggestionCommentRepo:      repository.NewSuggestionCommentRepository(queries),
		draftPageRepo:              repository.NewDraftPageRepository(queries),
		topicMemberRepo:            repository.NewTopicMemberRepository(queries),
		draftStamps:                stamps,
	}
}

// createSuggestionInput describes one suggestion to create.
//
// [Ja] createSuggestionInput は作成する編集提案 1 件の内容。
type createSuggestionInput struct {
	topic   *seededTopic
	creator *seededSpaceMember
	title   string
	body    string
	targets []suggestionTarget
	// status is where the suggestion has come to rest. Every one of them is
	// opened first and moved on from there, so this is the end of a history
	// rather than a column to be written.
	//
	// [Ja] status は編集提案が落ち着いた先。どの編集提案もまずオープンとして作られ、
	// そこから移っていくため、これは書き込む列ではなく履歴の終着点にあたる。
	status          model.SuggestionStatus
	renamesLastPage bool
	// comments is how long the discussion under the suggestion runs.
	//
	// [Ja] comments は、その編集提案の下に続く議論の長さ。
	comments    int
	editStarted bool
}

// suggestedPage is one page a suggestion proposes to change, after the proposed
// body has been rendered. What was rendered is kept because applying the
// suggestion writes the same title, body and links onto the page itself, and
// because the draft an edit in progress leaves behind holds a copy of them.
//
// [Ja] suggestedPage は、編集提案が変更を提案するページ 1 件を、提案された本文の
// レンダリング後の形で表す。レンダリング結果を保持するのは、編集提案の反映が同じ
// タイトル・本文・リンクをページ自体へ書き込むことと、進行中の編集が残す下書きが
// それらの写しを持つことによる。
type suggestedPage struct {
	id            model.SuggestionPageID
	target        suggestionTarget
	title         string
	body          string
	bodyHTML      string
	linkedPageIDs []model.PageID
}

// createSuggestion creates one suggestion together with the pages it proposes
// to change, the discussion under it and the mark of how it ended.
//
// The suggestion is opened first and moved on from there, because that is the
// only way production reaches the other statuses: a suggestion is created open,
// and applying or closing it is a later act on a suggestion that already
// exists.
//
// [Ja] createSuggestion は編集提案 1 件と、それが変更を提案するページ、その下に続く
// 議論、そしてどう終わったかの印を作成する。
//
// まずオープンとして作り、そこから移していくのは、本番が他のステータスに至る経路が
// それしかないため。編集提案はオープンとして作られ、反映やクローズは既に存在する
// 編集提案に対して後から行われる。
func (w *suggestionWriter) createSuggestion(ctx context.Context, input createSuggestionInput) error {
	if err := w.pages.ensureTopicInSpace(input.topic); err != nil {
		return err
	}

	number, err := w.suggestionRepo.GetNextNumber(ctx, w.space.id)
	if err != nil {
		return fmt.Errorf("次の編集提案番号の取得に失敗: %w", err)
	}

	suggestion, err := w.suggestionRepo.Create(ctx, repository.CreateSuggestionInput{
		SpaceID:              w.space.id,
		TopicID:              input.topic.id,
		CreatedSpaceMemberID: input.creator.id,
		Number:               number,
		Title:                input.title,
		Body:                 input.body,
		Status:               model.SuggestionStatusOpen,
	})
	if err != nil {
		return fmt.Errorf("編集提案 %s の作成に失敗: %w", input.title, err)
	}

	pages, err := w.createSuggestionPages(ctx, suggestion, input)
	if err != nil {
		return err
	}

	if err := w.createComments(ctx, suggestion, input); err != nil {
		return err
	}

	if input.editStarted {
		if err := w.startEdit(ctx, input, pages[0]); err != nil {
			return err
		}
	}

	return w.settle(ctx, suggestion, input, pages)
}

// createSuggestionPages creates the pages the suggestion proposes to change,
// each with the revision the proposal was written against and the revision the
// proposal itself is.
//
// The base revision is what the changed pages screen takes the diff against.
// Without it the screen has nothing to compare the proposal to and shows the
// whole body as added, so a page with no revision to point at is refused rather
// than proposed against.
//
// [Ja] createSuggestionPages は、編集提案が変更を提案するページを作成する。各ページは、
// 提案が書かれた時点のリビジョンと、提案そのものであるリビジョンを持つ。
//
// 基準リビジョンは、変更差分画面が差分を取る相手になる。これが無いと画面は提案を
// 比べる相手を持たず、本文全体を追加として表示するため、指すべきリビジョンが無い
// ページは提案の対象にせずエラーにする。
func (w *suggestionWriter) createSuggestionPages(
	ctx context.Context,
	suggestion *model.Suggestion,
	input createSuggestionInput,
) ([]suggestedPage, error) {
	pages := make([]suggestedPage, 0, len(input.targets))

	for i, target := range input.targets {
		title := target.page.title
		if input.renamesLastPage && i == len(input.targets)-1 {
			title += renamedPageTitleSuffix
		}
		body := suggestionPageBody(target.body)

		bodyHTML, linkedPageIDs, err := w.pages.render(ctx, createPageInput{
			topic:  input.topic,
			author: input.creator,
			title:  title,
			body:   body,
		})
		if err != nil {
			return nil, err
		}

		baseRevision, err := w.pages.pageRevisionRepo.FindLatestByPageID(ctx, target.page.id, w.space.id)
		if err != nil {
			return nil, fmt.Errorf("ページ %s の最新リビジョンの取得に失敗: %w", target.page.title, err)
		}
		if baseRevision == nil {
			return nil, fmt.Errorf("ページ %s に差分の基準となるリビジョンが無い", target.page.title)
		}

		suggestionPage, err := w.suggestionPageRepo.Create(ctx, repository.CreateSuggestionPageInput{
			SpaceID:        w.space.id,
			SuggestionID:   suggestion.ID,
			PageID:         target.page.id,
			PageRevisionID: &baseRevision.ID,
			Title:          &title,
			Body:           body,
			BodyHTML:       bodyHTML,
			LinkedPageIDs:  linkedPageIDs,
		})
		if err != nil {
			return nil, fmt.Errorf("編集提案 %s の提案ページの作成に失敗: %w", input.title, err)
		}

		if _, err := w.suggestionPageRevisionRepo.Create(ctx, repository.CreateSuggestionPageRevisionInput{
			SpaceID:             w.space.id,
			SuggestionPageID:    suggestionPage.ID,
			EditorSpaceMemberID: input.creator.id,
			Title:               &title,
			Body:                body,
			BodyHTML:            bodyHTML,
		}); err != nil {
			return nil, fmt.Errorf("編集提案 %s の提案ページのリビジョンの作成に失敗: %w", input.title, err)
		}

		pages = append(pages, suggestedPage{
			id:            suggestionPage.ID,
			target:        target,
			title:         title,
			body:          body,
			bodyHTML:      bodyHTML,
			linkedPageIDs: linkedPageIDs,
		})
	}

	return pages, nil
}

// createComments writes the discussion under a suggestion, with the two
// accounts taking turns.
//
// [Ja] createComments は編集提案の下に続く議論を書く。2 つのアカウントが交互に
// 発言する。
func (w *suggestionWriter) createComments(
	ctx context.Context,
	suggestion *model.Suggestion,
	input createSuggestionInput,
) error {
	for position := 1; position <= input.comments; position++ {
		number, err := w.suggestionCommentRepo.GetNextNumber(ctx, suggestion.ID)
		if err != nil {
			return fmt.Errorf("編集提案 %s の次のコメント番号の取得に失敗: %w", input.title, err)
		}

		author, err := suggestionCommentAuthor(w.space, position)
		if err != nil {
			return err
		}

		if _, err := w.suggestionCommentRepo.Create(ctx, repository.CreateSuggestionCommentInput{
			SpaceID:              w.space.id,
			SuggestionID:         suggestion.ID,
			CreatedSpaceMemberID: author.id,
			Number:               number,
			Body:                 suggestionCommentBody(position),
		}); err != nil {
			return fmt.Errorf("編集提案 %s のコメントの作成に失敗: %w", input.title, err)
		}
	}

	return nil
}

// startEdit leaves behind the draft that starting to edit a suggested page
// produces: a copy of what the suggestion proposes, linked to the suggestion
// page it came from.
//
// The editor writes such a draft when a member opens a suggested page for
// editing, and it is what the page detail screen reads to report that the page
// is being changed under a suggestion. The draft is left to roleOwner, because
// a draft belongs to the member who opened it and only one account can be
// looking at that screen.
//
// It takes the run's next stamp rather than the current time, which places it
// after every draft written before it. See draftStamps.
//
// [Ja] startEdit は、提案されたページの編集を始めたときに生まれる下書きを残す。
// 内容は編集提案が提案しているものの写しで、元になった編集提案ページに紐づく。
//
// この下書きは、メンバーが提案されたページを編集のために開いたときに編集画面が
// 書くものであり、ページ詳細画面が「このページは編集提案の下で変更されようとして
// いる」と知らせるために読むものでもある。下書きを roleOwner のものにしているのは、
// 下書きが開いたメンバーのものであり、その画面を見ているのは一方のアカウントに
// 限られるため。
//
// 打刻に現在時刻ではなく実行の次の打刻を使っているため、この下書きはそれ以前に
// 書かれたすべての下書きの後ろに並ぶ。draftStamps を参照。
func (w *suggestionWriter) startEdit(ctx context.Context, input createSuggestionInput, page suggestedPage) error {
	if _, err := w.draftPageRepo.Create(ctx, repository.CreateDraftPageInput{
		SpaceID:          w.space.id,
		PageID:           page.target.page.id,
		SpaceMemberID:    w.owner.id,
		TopicID:          input.topic.id,
		SuggestionPageID: &page.id,
		Title:            &page.title,
		Body:             page.body,
		BodyHTML:         page.bodyHTML,
		LinkedPageIDs:    page.linkedPageIDs,
		ModifiedAt:       w.draftStamps.next(),
	}); err != nil {
		return fmt.Errorf("編集提案 %s の編集中の下書きの作成に失敗: %w", input.title, err)
	}

	return nil
}

// settle moves the suggestion to where it came to rest. An open suggestion is
// already there, so only the two settled statuses have anything left to do.
//
// [Ja] settle は編集提案を、それが落ち着いた先へ移す。オープンな編集提案は既に
// そこにいるため、残る作業があるのは決着済みの 2 つのステータスだけになる。
func (w *suggestionWriter) settle(
	ctx context.Context,
	suggestion *model.Suggestion,
	input createSuggestionInput,
	pages []suggestedPage,
) error {
	update := repository.UpdateStatusInput{
		ID:      suggestion.ID,
		SpaceID: w.space.id,
		Status:  input.status,
	}

	switch input.status {
	case model.SuggestionStatusApplied:
		if err := w.applyToPages(ctx, input.topic, pages); err != nil {
			return err
		}
		appliedAt := time.Now()
		update.AppliedAt = &appliedAt
	case model.SuggestionStatusClosed:
	case model.SuggestionStatusOpen:
		return nil
	default:
		return fmt.Errorf("編集提案 %s に未知のステータス %d が指定された", input.title, input.status)
	}

	if _, err := w.suggestionRepo.UpdateStatus(ctx, update); err != nil {
		return fmt.Errorf("編集提案 %s のステータスの更新に失敗: %w", input.title, err)
	}

	return nil
}

// applyToPages writes what the suggestion proposes onto the pages themselves,
// which is what applying it does. The page then holds the proposal, keeps a
// revision of it, and counts the member who applied it among its editors.
//
// That member is roleOwner. Applying takes a scope only the administrator
// holds, so no other account could have been the one to do it.
//
// [Ja] applyToPages は、編集提案が提案している内容をページ自体へ書き込む。これが
// 反映の中身にあたる。ページは提案された内容を保持し、そのリビジョンを残し、反映した
// メンバーを編集者に数えるようになる。
//
// そのメンバーは roleOwner になる。反映には管理者だけが持つスコープが要るため、他の
// アカウントがそれを行ったということはありえないため。
func (w *suggestionWriter) applyToPages(ctx context.Context, topic *seededTopic, pages []suggestedPage) error {
	now := time.Now()

	for _, page := range pages {
		title := page.title

		if _, err := w.pages.pageRepo.Update(ctx, repository.UpdatePageInput{
			ID:            page.target.page.id,
			SpaceID:       w.space.id,
			TopicID:       topic.id,
			Title:         &title,
			Body:          page.body,
			BodyHTML:      page.bodyHTML,
			LinkedPageIDs: page.linkedPageIDs,
			ModifiedAt:    now,
			PublishedAt:   &now,
		}); err != nil {
			return fmt.Errorf("ページ %s への編集提案の反映に失敗: %w", title, err)
		}

		if _, err := w.pages.pageRevisionRepo.Create(ctx, repository.CreatePageRevisionInput{
			SpaceID:       w.space.id,
			SpaceMemberID: w.owner.id,
			PageID:        page.target.page.id,
			Title:         title,
			Body:          page.body,
			BodyHTML:      page.bodyHTML,
		}); err != nil {
			return fmt.Errorf("ページ %s のリビジョンの作成に失敗: %w", title, err)
		}

		pageEditor, err := w.pages.pageEditorRepo.FindOrCreate(ctx, repository.FindOrCreateInput{
			SpaceID:            w.space.id,
			PageID:             page.target.page.id,
			SpaceMemberID:      w.owner.id,
			LastPageModifiedAt: now,
		})
		if err != nil {
			return fmt.Errorf("ページ %s の編集者の登録に失敗: %w", title, err)
		}

		if _, err := w.pages.pageEditorRepo.UpdateLastPageModifiedAt(ctx, repository.UpdateLastPageModifiedAtInput{
			ID:                 pageEditor.ID,
			SpaceID:            w.space.id,
			LastPageModifiedAt: now,
		}); err != nil {
			return fmt.Errorf("ページ %s の編集者の更新に失敗: %w", title, err)
		}

		if err := w.topicMemberRepo.UpdateLastPageModifiedAt(
			ctx, w.space.id, topic.id, w.owner.id, now,
		); err != nil {
			return fmt.Errorf("トピック %s のメンバーの更新に失敗: %w", topic.name, err)
		}
	}

	return nil
}

// suggestionPageBody builds the body a suggestion proposes for a page: what the
// page holds now, with one line rewritten and a section added at the end.
//
// Keeping the rest of the body untouched is what gives the diff its unchanged
// context lines, and what makes the two changes readable as changes. A body
// written from scratch would be shown as the whole page being replaced, which
// says nothing about what was proposed.
//
// The rewritten line is the last one of the first paragraph that is neither
// blank nor a heading. A heading would move the change into the outline of the
// page, where a diff is read differently from a change made inside a paragraph.
// Appending to the last line of the paragraph rather than its first puts the
// sentence at the end of the paragraph. The seed writes a paragraph on one
// line, so that is usually the only line it has, and a body that does run over
// several lines still takes the sentence where a sentence ends.
//
// The sentence follows what the line already held with nothing between them,
// because these bodies are written in Japanese, where one sentence follows the
// next without a space.
//
// [Ja] suggestionPageBody は、編集提案がページに対して提案する本文を組み立てる。
// ページが現在保持している本文の 1 行を書き換え、末尾に節を足したもの。
//
// 残りの本文をそのままにするのは、差分に変更のない文脈行を与え、2 箇所の変更を
// 変更として読めるようにするため。一から書いた本文はページ全体の置き換えとして
// 表示され、何が提案されたのかを何も語らない。
//
// 書き換える行は、空行でも見出しでもない最初の段落の、最後の行になる。見出しを
// 書き換えると変更がページの見出し構成へ移り、そこでの差分は段落の中の変更とは違う
// 読まれ方をする。段落の最初の行ではなく最後の行に足すのは、文を段落の末尾に置く
// ため。シードは段落を 1 行で書くため通常はその 1 行しか無いが、複数行にまたがる
// 本文であっても文の切れ目に足せる。
//
// 文は、その行が元から持っていた内容との間に何も挟まずに続ける。これらの本文は
// 日本語で書かれており、日本語では文と文の間に空白を置かないため。
func suggestionPageBody(base string) string {
	lines := strings.Split(strings.TrimRight(base, "\n"), "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		last := i
		for last+1 < len(lines) && strings.TrimSpace(lines[last+1]) != "" {
			last++
		}
		lines[last] += suggestionRewrittenSentence

		break
	}

	return strings.Join(lines, "\n") + "\n" + suggestionAddedSection
}

// suggestionRewrittenSentence is appended to the one line the proposal
// rewrites, and suggestionAddedSection is what it adds at the end. Between them
// the diff holds both of the shapes a change takes: a line that was there and
// now reads differently, and lines that were not there at all.
//
// [Ja] suggestionRewrittenSentence は、提案が書き換える唯一の行に付け足される文。
// suggestionAddedSection は末尾に足される節。この 2 つにより、差分は変更が取る 2 つの
// 形をどちらも持つ。元からあって今は違う内容になった行と、元から無かった行である。
const (
	suggestionRewrittenSentence = "この文は、レビュー中の編集がこの行に足したものです。"

	suggestionAddedSection = `
## レビューメモ

この節は、このページが属する編集提案が足したものです。これにより差分は、書き換えられた行だけでなく、追加されたまとまりも見せるようになります。
`
)

// ordinarySuggestionBody says what a suggestion proposes. The suggestion body
// is shown as plain text rather than rendered, so it carries no markup: what is
// written here is what is read on the screen.
//
// The page title takes a space after it and none before. The titles the seed
// writes open with Japanese and close with a number, and a space belongs
// between a number and Japanese rather than between two Japanese characters.
//
// [Ja] ordinarySuggestionBody は、その編集提案が何を提案しているのかを述べる。編集提案の
// 本文はレンダリングされずプレーンテキストとして表示されるため、記法は書かない。
// ここに書いたものがそのまま画面で読まれる。
//
// ページタイトルの後ろには空白を置き、前には置かない。シードが書くタイトルは日本語で
// 始まり数字で終わっており、空白が要るのは数字と日本語の間であって、日本語と日本語の
// 間ではないため。
func ordinarySuggestionBody(pageTitle string) string {
	return fmt.Sprintf(`この編集提案は%s に小さな編集を提案します。1 行を書き換え、末尾に節を 1 つ足すものです。変更差分タブが差分を持ち、ページ自体は、反映されるまで最後に公開された内容を保ちます。`, pageTitle)
}

// suggestionCommentBodies are what the discussion under a suggestion is made
// of. They read as a conversation rather than as repeated filler, because a
// thread is looked at to see how remarks by two accounts sit against each
// other. The remarks answer one another in turn and the last one settles the
// thread, so they are written in the order they are posted in.
//
// There are as many of them as the longest discussion the amounts ask for.
// A shorter list would be cycled through, which would put the opening remark
// back under the one that settled the thread and undo the arc they are written
// as.
//
// [Ja] suggestionCommentBodies は、編集提案の下に続く議論の中身。同じ埋め草の
// 繰り返しではなく会話として読めるようにしている。スレッドは、2 つのアカウントの
// 発言が互いにどう並ぶかを見るために眺められるため。発言は順に互いへ答えており、
// 最後の 1 件がスレッドを収めるため、投稿される順に書いている。
//
// 件数は、件数の設定が求める最も長い議論に合わせている。これより短いと本文が循環し、
// スレッドを収めた発言の下に冒頭の発言が戻ってきて、発言が描いてきた流れが崩れる。
var suggestionCommentBodies = []string{
	`コメント %d。書き換えられた行は、いまページにあるものより読みやすくなっています。ただ、追加された節は、その上の段落が既に言っていることを繰り返しているように見えます。`,
	`コメント %d。節の件は同意です。要約を先に、補足を後ろに置く他のページに揃えられないでしょうか。`,
	`コメント %d。差分に「追加されたまとまり」を見せたかったので、いまは残してあります。反映する前に削って構いません。`,
	`コメント %d。もう 1 点あります。追加された節の見出しは、この編集提案が終われば意味を持たなくなります。オープンなうちに、本文に馴染む見出しへ変えておく価値はあるでしょうか。`,
	`コメント %d。このページが並ぶ他のページも同じ見出しを使っているので、ここだけ変えると浮いてしまいます。私はこのままでよいと思います。`,
	`コメント %d。私からは以上です。上の言い回しさえ決まれば、あとは反映してよい程度の小さな変更です。`,
}

// suggestionCommentBody writes the comment at the given position in the thread.
// The position is written into the body because the comments are all posted in
// the same moment, and the relative time each one is shown under cannot tell
// them apart.
//
// [Ja] suggestionCommentBody は、スレッドの指定位置のコメントを書く。位置を本文へ
// 書き込むのは、コメントがいずれも同じ瞬間に投稿され、各コメントの下に表示される
// 相対時刻ではそれらを見分けられないため。
func suggestionCommentBody(position int) string {
	return fmt.Sprintf(suggestionCommentBodies[(position-1)%len(suggestionCommentBodies)], position)
}

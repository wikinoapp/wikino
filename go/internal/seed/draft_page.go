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

// 一度も公開されていないページに付く下書きのタイトル。番号ではなく名前を
// 付けているのは、それぞれが固有の状態を見せるために存在することと、下書きの
// 一覧が下書き自身のタイトルで表示することによる。
const (
	unpublishedDraftTitle = "未公開ページの下書き"
	longHistoryDraftTitle = "履歴の長い下書き"
)

// ordinaryDraftRevisionsは、履歴を持たせる目的で書いたもの以外の下書きが
// 保存された回数。2は編集履歴に比較対象を与える最小の数である。下書きは保存に
// よってしか生まれないためリビジョンは最低1件あり、リビジョンの差分はその1つ
// 前との間で取られるため。
const ordinaryDraftRevisions = 2

// draftModifiedAtStepは、シードが2件の下書きのmodified_atの打刻の間に
// 空ける間隔。下書きを見せる2つの一覧はいずれもmodified_atの降順で並べるため、
// 複数の下書きを同じ時刻で打刻すると並び順がID任せになり、ホーム画面が残す下書きが
// 実行ごとに変わってしまう。
const draftModifiedAtStep = time.Minute

// draftStampsは、実行1回が書くすべての下書きのmodified_atの打刻を渡す。
// 各下書きは次の打刻を受け取り、実行を開始した時刻から遡っていくため、下書きは実行が
// 作成した順に一覧へ並ぶ。最初に作成したものが、最終更新の最も新しいものになる。
//
// カウンターはフェーズごとではなく実行全体で1つとする。未公開のページに付く下書きを
// 最初に作成しているのは、新しいものを数件しか残さないホーム画面がそれらと出会う場に
// なるようにするためである。後続のフェーズがそれぞれのtime.Now() で打刻すると、
// 下書きを書いたフェーズより後に走る分だけそれらより新しくなり、その画面から
// 押し出してしまう。
type draftStamps struct {
	origin time.Time
	issued int
}

// newDraftStampsはoriginから遡っていく打刻を返す。
func newDraftStamps(origin time.Time) *draftStamps {
	return &draftStamps{origin: origin}
}

// nextは、次に作成する下書きの打刻を返す。
func (s *draftStamps) next() time.Time {
	at := s.origin.Add(-time.Duration(s.issued) * draftModifiedAtStep)
	s.issued++

	return at
}

// newPageDraftSpecは、一度も公開されていないページに対して書かれた下書き
// 1件の内容。これは、ページ作成ボタンを押してから公開するまでの間に編集画面が
// 残すものにあたる。
type newPageDraftSpec struct {
	// titleは下書きの呼び名で、一度もタイトルを付けられていない下書きでは
	// nilになる。下書きはページとは別に自身のタイトルを持ち、一覧はページの
	// タイトル、次いで「無題」の表示へフォールバックする。タイトルの無い下書きは、
	// そのフォールバックの末端を確認できる唯一の手段になる。
	title *string
	// introは本文の書き出し。その後ろは保存が書き足す。
	intro string
	// longHistoryは、他の下書きのように数回ではなく、件数の設定が求める
	// 回数だけ保存されることを求める。
	longHistory bool
}

// newPageDraftSpecsは未公開のページに付く下書き。全体で、ページが一度も
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

// generateDraftPagesは、ホーム画面・下書き一覧画面・編集画面の2つの
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
			"ownerの下書き %d件は、未公開のページに付ける %d件を下回っている",
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

	// 下書きは両アカウントが読めるトピックへ分散させる。下書き一覧画面は
	// スペースとトピックでグループ分けするため、1つのトピックに寄せるとその画面が
	// 1グループになり、グループ分けを見分けるものが無くなる。
	draftTopics := []*seededTopic{topics.handbook, topics.notes, topics.privateNotes}

	targets, err := collectDraftTargets(ctx, dbtx, draftTopics, onPublishedPages+amt.collaboratorDraftPages)
	if err != nil {
		return err
	}

	writer := newDraftWriter(dbtx, spaces.wiki)

	// 未公開のページに付く下書きを先に作成し、最終更新が最も新しい状態に
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

	// 2つのアカウントはそれぞれ別のページに対して下書きを書く。同じページを
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

// draftTargetは、下書きが対象とする公開済みのページと、それが属する
// トピック。トピックをページと一緒に持つのは、下書き自身がトピックを持つことと、
// 下書きの本文のレンダリングに、本文が読まれるトピックの名前が要ることによる。
type draftTarget struct {
	topic *seededTopic
	page  *seededPage
}

// collectDraftTargetsは、下書きが対象とする公開済みのページを選ぶ。
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
				"トピック %sの下書き対象にできるページが %d件しかなく、必要な %d件に足りない",
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

// listDraftTargetPagesは、あるトピックのうち下書きの対象にできるページを
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
		return nil, fmt.Errorf("トピック %sの公開済みページの取得に失敗: %w", topic.name, err)
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
			return nil, fmt.Errorf("トピック %sの公開済みページの読み取りに失敗: %w", topic.name, err)
		}

		page := &seededPage{id: model.PageID(id), number: model.PageNumber(number)}
		if title != nil {
			page.title = *title
		}
		targets = append(targets, draftTarget{topic: topic, page: page})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("トピック %sの公開済みページの読み取りに失敗: %w", topic.name, err)
	}

	return targets, nil
}

// draftWriterは1つのスペースに下書きとそのリビジョンを作成する。自前の
// pageWriterを持つのは、一度も公開されていないページの下書きには、まずその
// ページの作成が必要であることと、下書きの本文がページの本文と同じ経路で
// レンダリングされることによる。
type draftWriter struct {
	space                 *seededSpace
	pages                 *pageWriter
	draftPageRepo         *repository.DraftPageRepository
	draftPageRevisionRepo *repository.DraftPageRevisionRepository
}

// newDraftWriterはspaceに下書きを作成するwriterを返す。
func newDraftWriter(dbtx query.DBTX, space *seededSpace) *draftWriter {
	queries := query.New(dbtx)

	return &draftWriter{
		space:                 space,
		pages:                 newPageWriter(dbtx, space),
		draftPageRepo:         repository.NewDraftPageRepository(queries),
		draftPageRevisionRepo: repository.NewDraftPageRevisionRepository(queries),
	}
}

// newPageDraftInputは、一度も公開されていないページに付く下書き1件の内容。
type newPageDraftInput struct {
	topic      *seededTopic
	member     *seededSpaceMember
	spec       newPageDraftSpec
	revisions  int
	modifiedAt time.Time
}

// createNewPageDraftは、ページ作成ボタンが残すページと、それに対して
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

// createPublishedPageDraftは、既に公開されているページの下書きを作成する。
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

// createDraftInputは作成する下書き1件の内容。
type createDraftInput struct {
	topic  *seededTopic
	member *seededSpaceMember
	page   *seededPage
	title  *string
	intro  string
	// revisionsは下書きが保存された回数。下書きは最後の保存が書いたものを
	// 保持し、編集履歴はそのすべてを保持する。
	revisions  int
	modifiedAt time.Time
}

// labelはエラーメッセージ内で下書きを名指しする。一度もタイトルを
// 付けられていない下書きには、他に名指しするものが無いため。
func (input createDraftInput) label() string {
	if input.title != nil && *input.title != "" {
		return *input.title
	}

	return "タイトル未設定の下書き"
}

// createDraftは下書き1件と、その各保存が残したリビジョンを作成する。
// これは編集画面から保存したときに残る行の一式にあたる。
//
// どちらもRepositoryを経由する。下書きの保存は既にGo側が担当しているため、
// 画面が呼ぶCreateをそのままシードも呼べる。
func (w *draftWriter) createDraft(ctx context.Context, input createDraftInput) error {
	if err := w.pages.ensureTopicInSpace(input.topic); err != nil {
		return err
	}

	var title string
	if input.title != nil {
		title = *input.title
	}

	// 保存のたびに1行を書き足す。これにより履歴が、同じ本文の保存の
	// 繰り返しではなく、時間をかけて書かれていく本文として読め、どのリビジョンの
	// 差分もその保存が足した1行を見せるようになる。
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
		return fmt.Errorf("下書き %sの作成に失敗: %w", input.label(), err)
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
			return fmt.Errorf("下書き %sのリビジョンの作成に失敗: %w", input.label(), err)
		}
	}

	return nil
}

// draftRevisionBodyは、指定の保存を終えた時点で下書きが持っていた本文を
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

// publishedPageDraftIntroは、既に公開されているページに対して書かれた
// 下書きの本文の書き出しを組み立てる。タイトルは日本語の文の中に書き込まれるため、
// その始まりと終わりの文字によって空白を補う。
func publishedPageDraftIntro(pageTitle string) string {
	return fmt.Sprintf(`これは%sの未公開の編集です。下書きは、公開されるまでページ本体とは別に保持されます。そのためページ側は最後に公開された内容を見せたままで、編集画面が代わりに開くのがこの下書きになります。`, spacedInJapanese(pageTitle))
}

// spacedInJapaneseは、前後を日本語の文に挟まれたsのうち、ASCII文字で接する側に
// 空白を足して返す。
//
// 空白の要否は接する2つの文字で決まり、文字列全体では決まらない。下書きが対象と
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

// Package seed populates a development database with the data browser
// verification needs. It creates accounts, spaces, topics, the pages that
// screens such as pagination cannot be checked without, the drafts those pages
// are edited through, and the suggestions by which an edit is proposed rather
// than published.
//
// Rows are written through the existing repositories wherever production
// already has a Create, and through INSERT statements kept inside this package
// wherever it does not. Seeding is not a reason to grow the Infrastructure
// layer with code that only the seed calls.
//
// Seeded prose bodies are written one line per paragraph. Markdown page bodies
// render a line break inside a paragraph as a <br>, while suggestion bodies and
// comments preserve it through white-space: pre-wrap. A body wrapped by hand
// therefore shows those breaks on the screen instead of the wrapping the
// browser does at the width the body is read in. bodies/markdown-guide.md is the
// exception: it is a Markdown file, and the shared Markdown linter asks those
// for one sentence per line.
//
// [Ja] seed パッケージは開発用データベースへ、ブラウザ確認に必要なデータを投入する。
// アカウント・スペース・トピックと、ページネーションのように一定のデータが無いと
// 確認できない画面のためのページ、それらのページを編集するための下書き、そして
// 公開ではなく提案として編集を行う編集提案を作成する。
//
// 行の書き込みには、本番用の Create が既にある対象では既存の Repository を使い、
// 無い対象では本パッケージ内に閉じた INSERT を使う。シードのために、シードだけが
// 呼ぶコードを Infrastructure 層へ増やすことはしない。
//
// シードが書く地の文は 1 段落 1 行で書く。Markdown のページ本文は段落内の改行を
// <br> として描画し、編集提案の本文とコメントは white-space: pre-wrap によって改行を
// 保つ。手で折り返した本文は、本文が読まれる幅でブラウザが行う折り返しではなく、その
// 改行を画面に見せることになる。bodies/markdown-guide.md は例外とする。Markdown
// ファイルであり、共通の Markdown リンタが句点改行を求めるため。
package seed

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/wikinoapp/wikino/go/internal/config"
)

// Runner performs one seeding run against db.
//
// [Ja] Runner は db に対するシード実行 1 回分を受け持つ。
type Runner struct {
	db  *sql.DB
	cfg *config.Config
	out io.Writer
}

// NewRunner returns a Runner that writes its progress to out.
//
// [Ja] NewRunner は進捗を out に書く Runner を返す。
func NewRunner(db *sql.DB, cfg *config.Config, out io.Writer) *Runner {
	return &Runner{db: db, cfg: cfg, out: out}
}

// amounts is how much of each kind of data a run creates. The counts are held
// in one struct rather than written where they are used so that what the seed
// produces can be read in one place, and so that tests can ask for a few rows
// where a run asks for hundreds.
//
// [Ja] amounts は、実行 1 回分が各種データを何件作るか。件数を使う場所へ書かず
// 1 つの構造体にまとめているのは、シードが何を作るのかを 1 箇所で読めるように
// するためと、実行が数百件を求める場所でテストが数件を求められるようにするため。
type amounts struct {
	// handbookPages fills topics.handbook. 250 pages is 3 pages of the
	// topic's listing at 100 per page, the last of them holding 50: the first,
	// a middle and a partial last page can all be seen.
	//
	// [Ja] handbookPages は「ハンドブック」トピックを埋める。250 ページは、1 ページ
	// 100 件のトピック一覧で 3 ページ分にあたり、最終ページが 50 件の端数になる。
	// 最初・途中・端数の最終ページのいずれも確認できる。
	handbookPages int
	// privateNotesPages and secretPages give the private topics pages of their
	// own, so that a private topic can be checked for what it holds and not
	// only for whether it is listed. Most of them go to topics.privateNotes, the
	// topic both accounts have joined, because those are the pages both
	// accounts can open.
	//
	// Together with handbookPages they also fill the space-wide listing. The
	// exact total is not settled here — later generators publish pages of
	// their own — so what these counts hold to is a total between 301 and 400,
	// which at 100 per page is 4 pages with a remainder on the last.
	//
	// [Ja] privateNotesPages と secretPages は、非公開トピックにもページを与える。
	// 非公開トピックについて、一覧に出るかどうかだけでなく、中に何があるかも
	// 確認できるようにするため。多くを「非公開ノート」に置くのは、両アカウントが
	// 参加しているトピックであり、そこのページが両方から開けるものになるため。
	//
	// handbookPages と合わせて、スペース全体の一覧も埋める。正確な合計はここでは
	// 決まらない (後続の生成器も自前の公開済みページを追加するため) ため、これらの
	// 件数が守るのは合計が 301〜400 に収まること。1 ページ 100 件ならこれは 4 ページ
	// 分にあたり、最終ページが端数になる。
	privateNotesPages int
	secretPages       int
	// linkHubTargets, linkHubBacklinks and nestedBacklinks size the three
	// listings that appear under a page body. Each one paginates on its own and
	// at its own size, so each needs a count of its own to leave a partial last
	// page: 50 links at 15 per page is 4 pages with 5 on the last, 45 backlinks
	// at 14 is 4 pages with 3, and 20 nested backlinks at 13 is 2 pages with 7.
	//
	// [Ja] linkHubTargets・linkHubBacklinks・nestedBacklinks は、ページ本文の下に
	// 並ぶ 3 つの一覧の大きさを決める。3 つはそれぞれ独立に、しかも異なる件数で
	// ページングするため、端数の最終ページを作るにはそれぞれに件数が要る。リンクは
	// 1 ページ 15 件で 50 件なら 4 ページ (端数 5)、バックリンクは 1 ページ 14 件で
	// 45 件なら 4 ページ (端数 3)、ネストしたバックリンクは 1 ページ 13 件で 20 件
	// なら 2 ページ (端数 7) になる。
	linkHubTargets   int
	linkHubBacklinks int
	nestedBacklinks  int
	// pinnedPages and trashedPages are the two states a page is put into after
	// it has been published. Neither one appears in the regular page listings —
	// pinned pages are shown above them, trashed pages not at all — so these
	// counts are small and stay out of the counts chosen above: a handful is
	// enough for the pinned section to have an order worth reading and for the
	// trash to hold a list rather than a single row.
	//
	// [Ja] pinnedPages と trashedPages は、公開後のページが置かれる 2 つの状態。
	// どちらも通常のページ一覧には出ない (ピン留めはその上に表示され、ゴミ箱は
	// まったく出ない) ため、件数は小さく、上で選んだ件数にも影響しない。ピン留めの
	// 区画に読み取れる順序があり、ゴミ箱が 1 行ではなく一覧になるには、数件あれば
	// 足りる。
	pinnedPages  int
	trashedPages int
	// soloNotesPages and soloSecretPages fill the two topics of seed-solo, the
	// space only roleOwner has joined. Neither count is chosen to page a
	// listing: this space is opened to see what is readable from outside it, and
	// a handful of pages is enough for the public topic to be entered and for
	// the private one to be a listing rather than a single row.
	//
	// The public topic is given the larger of the two because it is the one that
	// gets browsed from every account. The private topic is only ever opened by
	// roleOwner; from an account outside the space it is a URL that has to be
	// not found.
	//
	// [Ja] soloNotesPages と soloSecretPages は、roleOwner だけが参加しているスペース
	// seed-solo の 2 つのトピックを埋める。どちらの件数も一覧をページ送りさせる
	// ために選んだものではない。このスペースは、外から何が読めるのかを見るために
	// 開くものであり、公開トピックに入っていけること、非公開トピックが 1 行では
	// なく一覧になることには数件で足りる。
	//
	// 公開トピックのほうを多くしているのは、すべてのアカウントから閲覧される
	// トピックであるため。非公開トピックを開くのは roleOwner だけで、スペースの外の
	// アカウントから見たそれは、見つからない必要のある URL でしかない。
	soloNotesPages  int
	soloSecretPages int
	// ownerDraftPages and collaboratorDraftPages are how many drafts each account
	// keeps. A draft belongs to the member who wrote it, so the listings that
	// show them are filled per account rather than once for the space.
	//
	// Both counts clear the five drafts the home screen shows, so that screen
	// holds a full row and drops the rest. roleOwner's also clears the twenty the
	// editor's draft column shows, which leaves that column with a draft it does
	// not fit; roleCollaborator is left below it, so the column can be seen both
	// full and not.
	//
	// [Ja] ownerDraftPages と collaboratorDraftPages は、各アカウントが持つ下書きの
	// 件数。下書きは書いたメンバーのものであるため、それを見せる一覧はスペースに
	// 1 つではなくアカウントごとに埋まる。
	//
	// どちらの件数もホーム画面が見せる 5 件を超えており、あの画面は 1 行分が
	// 埋まって残りが落ちる。roleOwner の件数はさらに、編集画面の下書きカラムが
	// 見せる 20 件も超えるため、あのカラムには収まらない下書きが生まれる。
	// roleCollaborator はその手前に留めており、カラムが埋まり切る状態と切らない
	// 状態の両方を確認できる。
	ownerDraftPages        int
	collaboratorDraftPages int
	// draftRevisions is how many times the one draft written to have a history
	// has been saved. It clears the twenty revisions the editor's edit history
	// shows, so the oldest entry on that column is not the first save, which is
	// what tells a list cut short by the limit apart from a complete one.
	//
	// [Ja] draftRevisions は、履歴を持たせる目的で書いた 1 件の下書きが保存された
	// 回数。編集画面の編集履歴が見せる 20 件を超えるため、あのカラムの末尾が最初の
	// 保存にならない。これにより、上限で切られた一覧と全件が並ぶ一覧を見分けられる。
	draftRevisions int
	// openSuggestions, appliedSuggestions and closedSuggestions are how many
	// suggestions each of the two tabs of a topic's suggestion listing holds.
	// The listing does not paginate, so the open tab is filled until it runs
	// past the height of a screen rather than to a page's worth of rows. The
	// closed tab is given fewer, and both of the statuses that land in it.
	//
	// openSuggestions counts the suggestions written to show a state of their
	// own along with the rest, because the listing does not tell them apart.
	//
	// [Ja] openSuggestions・appliedSuggestions・closedSuggestions は、トピックの
	// 編集提案一覧の 2 つのタブがそれぞれ何件を持つか。この一覧はページングしない
	// ため、オープンのタブは 1 ページ分の行数ではなく画面の高さを超える件数まで
	// 埋める。クローズのタブは件数を少なくし、そこに入る 2 つのステータスを両方とも
	// 持たせる。
	//
	// openSuggestions は、固有の状態を見せるために書いた編集提案も併せて数える。
	// 一覧がそれらを区別しないため。
	openSuggestions    int
	appliedSuggestions int
	closedSuggestions  int
	// suggestionComments is how long the discussion under the one suggestion
	// written to carry a thread runs. The comments are all shown at once, so
	// the count is what makes the thread longer than the suggestion above it
	// instead of a remark or two below it.
	//
	// [Ja] suggestionComments は、スレッドを持たせる目的で書いた 1 件の編集提案の
	// 下に続く議論の長さ。コメントは一度にすべて表示されるため、この件数によって
	// 議論が、下に 1、2 件の発言が付いた状態ではなく、上にある編集提案より長い
	// ものになる。
	suggestionComments int
}

// defaultAmounts is what a run creates. Tests pass their own amounts.
//
// [Ja] defaultAmounts は実行 1 回分が作る件数。テストは自前の件数を渡す。
var defaultAmounts = amounts{
	handbookPages:          250,
	privateNotesPages:      60,
	secretPages:            10,
	linkHubTargets:         50,
	linkHubBacklinks:       45,
	nestedBacklinks:        20,
	pinnedPages:            3,
	trashedPages:           3,
	soloNotesPages:         5,
	soloSecretPages:        3,
	ownerDraftPages:        22,
	collaboratorDraftPages: 6,
	draftRevisions:         24,
	openSuggestions:        20,
	appliedSuggestions:     4,
	closedSuggestions:      4,
	suggestionComments:     6,
}

// state carries what each generator produced to the generators that follow it,
// along with what the run laid out before the first one.
//
// [Ja] state は各生成器が作ったものを、後続の生成器へ引き渡す。実行が最初の生成器の
// 前に用意したものも合わせて運ぶ。
type state struct {
	users  *seededUsers
	spaces *seededSpaces
	topics *seededTopics
	// draftStamps is shared by every generator that writes drafts, so that the
	// drafts of a run are ordered by the order the run created them rather than
	// by the phase each one belongs to.
	//
	// [Ja] draftStamps は下書きを書くすべての生成器で共有する。実行 1 回分の下書きの
	// 並びが、それぞれが属するフェーズではなく、実行が作成した順で決まるようにするため。
	draftStamps *draftStamps
}

// generator is one named step of a run. Holding the steps as a list keeps the
// order in one place and lets the phase numbers be counted off at run time,
// instead of being written into comments that drift from what actually runs.
//
// [Ja] generator は実行 1 回分の名前付きステップ。ステップを一覧として持つことで
// 順序が 1 箇所にまとまり、フェーズ番号を実行時に採番できる。番号をコメントへ
// 書き込むと、実際に走る内容とずれていくため。
type generator struct {
	name string
	run  func(ctx context.Context, st *state) error
}

// Run empties the database and generates the seed data.
//
// [Ja] Run はデータベースを空にしてシードデータを生成する。
func (r *Runner) Run(ctx context.Context) error {
	if err := EnsureDevEnv(r.cfg.Env); err != nil {
		return err
	}

	// Read the roster before touching the database: a roster that cannot be
	// read is a configuration mistake, and it should surface before anything
	// has been deleted.
	//
	// [Ja] データベースへ触れる前に名簿を読む。名簿を読めないのは設定の誤りであり、
	// 何かを削除する前に表面化させたいため。
	roster, err := loadUserRoster(rosterPath)
	if err != nil {
		return err
	}

	if err := r.db.PingContext(ctx); err != nil {
		return fmt.Errorf("データベースへのpingに失敗: %w", err)
	}

	// Report which database is about to be emptied, and which roster the
	// accounts come from. The command destroys every row it manages, so the
	// developer gets to see the target before the deletion rather than after
	// it. The roster is named beside it because which accounts a run creates
	// depends on a file that is not in version control.
	//
	// [Ja] これから空にするデータベースと、アカウントの供給元となる名簿を報告する。
	// 本コマンドは管理対象の行をすべて破棄するため、削除後ではなく削除前に対象を
	// 目視できるようにする。名簿を並べて出すのは、実行がどのアカウントを作るのかが、
	// バージョン管理に入っていないファイルに依存しているため。
	dbName, err := currentDatabase(ctx, r.db)
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "シードデータを投入します", "database", dbName, "users_file", roster.path)

	startedAt := time.Now()

	if err := cleanup(ctx, r.db); err != nil {
		return err
	}
	slog.InfoContext(ctx, "既存データをクリーンアップしました", "table_count", len(cleanupTables))

	generators := []generator{
		{name: "ユーザー", run: func(ctx context.Context, st *state) error {
			users, err := generateUsers(ctx, r.db, r.out, roster)
			if err != nil {
				return err
			}
			st.users = users

			return nil
		}},
		{name: "スペース", run: func(ctx context.Context, st *state) error {
			spaces, err := generateSpaces(ctx, r.db, r.out, st.users)
			if err != nil {
				return err
			}
			st.spaces = spaces

			return nil
		}},
		{name: "トピック", run: func(ctx context.Context, st *state) error {
			topics, err := generateTopics(ctx, r.db, r.out, st.spaces)
			if err != nil {
				return err
			}
			st.topics = topics

			return nil
		}},
		{name: "Markdown記法紹介ページ", run: func(ctx context.Context, st *state) error {
			return generateMarkdownGuide(ctx, r.db, r.out, st.spaces, st.topics)
		}},
		{name: "ページネーション用ページ", run: func(ctx context.Context, st *state) error {
			return generateBulkPages(ctx, r.db, r.out, defaultAmounts, st.spaces, st.topics)
		}},
		{name: "リンク集中ページ", run: func(ctx context.Context, st *state) error {
			return generateLinkHub(ctx, r.db, r.out, defaultAmounts, st.spaces, st.topics)
		}},
		{name: "状態バリエーションページ", run: func(ctx context.Context, st *state) error {
			return generatePageVariations(ctx, r.db, r.out, defaultAmounts, st.spaces, st.topics)
		}},
		{name: "表示崩れ確認用ページ", run: func(ctx context.Context, st *state) error {
			return generateSandboxPages(ctx, r.db, r.out, st.spaces, st.topics)
		}},
		{name: "個人スペースのページ", run: func(ctx context.Context, st *state) error {
			return generateSoloPages(ctx, r.db, r.out, defaultAmounts, st.spaces, st.topics)
		}},
		{name: "下書きページ", run: func(ctx context.Context, st *state) error {
			return generateDraftPages(ctx, r.db, r.out, defaultAmounts, st.spaces, st.topics, st.draftStamps)
		}},
		{name: "編集提案", run: func(ctx context.Context, st *state) error {
			return generateSuggestions(ctx, r.db, r.out, defaultAmounts, st.spaces, st.topics, st.draftStamps)
		}},
	}

	st := &state{draftStamps: newDraftStamps(startedAt)}
	for i, g := range generators {
		slog.InfoContext(ctx, fmt.Sprintf("フェーズ %d/%d: %sを生成します", i+1, len(generators), g.name))
		if err := g.run(ctx, st); err != nil {
			return fmt.Errorf("%sの生成に失敗: %w", g.name, err)
		}
	}

	// The accounts are named only when a generator actually produced them.
	// state's fields are nil until the step that fills them has run. Generators
	// are kept in dependency order and may be extended, so the completion log must
	// not assume that any particular step took place.
	//
	// Each account is logged under the role a generator names it by, so that the
	// address to sign in with can be read off the line that says which role it
	// belongs to.
	//
	// [Ja] アカウントは、生成器が実際に作った場合にだけ出力する。state の
	// フィールドは、それを埋めるステップが走るまで nil である。生成器は依存順に
	// 並べられ、今後追加されるため、完了ログが特定のステップの実行を前提に
	// してはならない。
	//
	// 各アカウントは、生成器がそれを名指しする役割の名前で出力する。どの役割の
	// ものかを示す行から、サインインに使うアドレスを読み取れるようにするため。
	attrs := []any{"elapsed", time.Since(startedAt).Round(time.Millisecond)}
	if st.users != nil {
		for _, account := range roster.users {
			if user := st.users.user(account.role); user != nil {
				attrs = append(attrs, string(account.role), user.Email)
			}
		}
	}
	slog.InfoContext(ctx, "シードデータの投入が完了しました", attrs...)

	return nil
}

// EnsureDevEnv rejects any environment other than development.
//
// The seed deletes every row it manages and creates accounts whose password
// the roster chooses, and the credentials helper prints that password to
// whoever asks. Against anything but a development database, both have to be
// impossible rather than merely discouraged.
//
// It takes the environment name rather than a *config.Config so that a command
// can apply the same check to the raw APP_ENV, before config.Load substitutes
// its development default for an unset value. Every guard calls this one
// function, so the wording of the refusal cannot drift apart, and the refusal
// names what it covers rather than the seed alone.
//
// [Ja] EnsureDevEnv は開発環境以外での実行を拒否する。
//
// シードは管理対象の行をすべて削除し、名簿が決めたパスワードでサインインできる
// アカウントを作る。資格情報のヘルパーは、そのパスワードを尋ねた相手へ出力する。
// 開発用以外のデータベースに対しては、どちらも推奨しないのではなく実行できない
// ようにする必要がある。
//
// *config.Config ではなく環境名を受け取るのは、config.Load が未設定時の既定値を
// 補う前の生の APP_ENV に対して、コマンド側が同じ検査を適用できるようにするため。
// すべてのガードがこの 1 つの関数を呼ぶため、拒否の文言がずれることがなく、文言も
// シードだけでなく対象全体を名指しする形にしている。
func EnsureDevEnv(env string) error {
	if env != "dev" {
		return fmt.Errorf("開発用データを扱うコマンドは開発環境でのみ実行できます (APP_ENV=%s)", env)
	}

	return nil
}

// currentDatabase returns the name of the database the connection is bound to.
//
// [Ja] currentDatabase は接続先データベースの名前を返す。
func currentDatabase(ctx context.Context, db *sql.DB) (string, error) {
	var name string
	if err := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&name); err != nil {
		return "", fmt.Errorf("接続先データベース名の取得に失敗: %w", err)
	}

	return name, nil
}

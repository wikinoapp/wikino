// Package seedは開発用データベースへ、ブラウザ確認に必要なデータを投入する。
// アカウント・スペース・トピックと、ページネーションのように一定のデータが無いと
// 確認できない画面のためのページ、それらのページを編集するための下書き、そして
// 公開ではなく提案として編集を行う編集提案を作成する。
//
// 行の書き込みには、本番用のCreateが既にある対象では既存のRepositoryを使い、
// 無い対象では本パッケージ内に閉じたINSERTを使う。シードのために、シードだけが
// 呼ぶコードをInfrastructure層へ増やすことはしない。
//
// シードが書く地の文は1段落1行で書く。Markdownのページ本文は段落内の改行を
// <br> として描画し、編集提案の本文とコメントはwhite-space: pre-wrapによって改行を
// 保つ。手で折り返した本文は、本文が読まれる幅でブラウザが行う折り返しではなく、その
// 改行を画面に見せることになる。この規則はbodies/ 配下の本文にも及ぶ。そこにある
// ファイルを1つも .mdと名付けていないのはそのためで、共通のMarkdownリンタが
// .mdを一律に句点改行へ揃えてしまうため。
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

// Runnerはdbに対するシード実行1回分を受け持つ。
type Runner struct {
	db  *sql.DB
	cfg *config.Config
	out io.Writer
}

// NewRunnerは進捗をoutに書くRunnerを返す。
func NewRunner(db *sql.DB, cfg *config.Config, out io.Writer) *Runner {
	return &Runner{db: db, cfg: cfg, out: out}
}

// amountsは、実行1回分が各種データを何件作るか。件数を使う場所へ書かず
// 1つの構造体にまとめているのは、シードが何を作るのかを1箇所で読めるように
// するためと、実行が数百件を求める場所でテストが数件を求められるようにするため。
type amounts struct {
	// handbookPagesは「ハンドブック」トピックを埋める。250ページは、1ページ
	// 100件のトピック一覧で3ページ分にあたり、最終ページが50件の端数になる。
	// 最初・途中・端数の最終ページのいずれも確認できる。
	handbookPages int
	// privateNotesPagesとsecretPagesは、非公開トピックにもページを与える。
	// 非公開トピックについて、一覧に出るかどうかだけでなく、中に何があるかも
	// 確認できるようにするため。多くを「非公開ノート」に置くのは、両アカウントが
	// 参加しているトピックであり、そこのページが両方から開けるものになるため。
	//
	// handbookPagesと合わせて、スペース全体の一覧も埋める。正確な合計はここでは
	// 決まらない (後続の生成器も自前の公開済みページを追加するため) ため、これらの
	// 件数が守るのは合計が301〜400に収まること。1ページ100件ならこれは4ページ
	// 分にあたり、最終ページが端数になる。
	privateNotesPages int
	secretPages       int
	// linkHubTargets・linkHubBacklinks・nestedBacklinksは、ページ本文の下に
	// 並ぶ3つの一覧の大きさを決める。3つはそれぞれ独立に、しかも異なる件数で
	// ページングするため、端数の最終ページを作るにはそれぞれに件数が要る。リンクは
	// 1ページ15件で50件なら4ページ (端数5)、バックリンクは1ページ14件で
	// 45件なら4ページ (端数3)、ネストしたバックリンクは1ページ13件で20件
	// なら2ページ (端数7) になる。
	linkHubTargets   int
	linkHubBacklinks int
	nestedBacklinks  int
	// pinnedPagesとtrashedPagesは、公開後のページが置かれる2つの状態。
	// どちらも通常のページ一覧には出ない (ピン留めはその上に表示され、ゴミ箱は
	// まったく出ない) ため、件数は小さく、上で選んだ件数にも影響しない。ピン留めの
	// 区画に読み取れる順序があり、ゴミ箱が1行ではなく一覧になるには、数件あれば
	// 足りる。
	pinnedPages  int
	trashedPages int
	// soloNotesPagesとsoloSecretPagesは、roleOwnerだけが参加しているスペース
	// seed-soloの2つのトピックを埋める。どちらの件数も一覧をページ送りさせる
	// ために選んだものではない。このスペースは、外から何が読めるのかを見るために
	// 開くものであり、公開トピックに入っていけること、非公開トピックが1行では
	// なく一覧になることには数件で足りる。
	//
	// 公開トピックのほうを多くしているのは、すべてのアカウントから閲覧される
	// トピックであるため。非公開トピックを開くのはroleOwnerだけで、スペースの外の
	// アカウントから見たそれは、見つからない必要のあるURLでしかない。
	soloNotesPages  int
	soloSecretPages int
	// ownerDraftPagesとcollaboratorDraftPagesは、各アカウントが持つ下書きの
	// 件数。下書きは書いたメンバーのものであるため、それを見せる一覧はスペースに
	// 1つではなくアカウントごとに埋まる。
	//
	// どちらの件数もホーム画面が見せる5件を超えており、あの画面は1行分が
	// 埋まって残りが落ちる。roleOwnerの件数はさらに、編集画面の下書きカラムが
	// 見せる20件も超えるため、あのカラムには収まらない下書きが生まれる。
	// roleCollaboratorはその手前に留めており、カラムが埋まり切る状態と切らない
	// 状態の両方を確認できる。
	ownerDraftPages        int
	collaboratorDraftPages int
	// draftRevisionsは、履歴を持たせる目的で書いた1件の下書きが保存された
	// 回数。編集画面の編集履歴が見せる20件を超えるため、あのカラムの末尾が最初の
	// 保存にならない。これにより、上限で切られた一覧と全件が並ぶ一覧を見分けられる。
	draftRevisions int
	// openSuggestions・appliedSuggestions・closedSuggestionsは、トピックの
	// 編集提案一覧の2つのタブがそれぞれ何件を持つか。この一覧はページングしない
	// ため、オープンのタブは1ページ分の行数ではなく画面の高さを超える件数まで
	// 埋める。クローズのタブは件数を少なくし、そこに入る2つのステータスを両方とも
	// 持たせる。
	//
	// openSuggestionsは、固有の状態を見せるために書いた編集提案も併せて数える。
	// 一覧がそれらを区別しないため。
	openSuggestions    int
	appliedSuggestions int
	closedSuggestions  int
	// suggestionCommentsは、スレッドを持たせる目的で書いた1件の編集提案の
	// 下に続く議論の長さ。コメントは一度にすべて表示されるため、この件数によって
	// 議論が、下に1、2件の発言が付いた状態ではなく、上にある編集提案より長い
	// ものになる。
	suggestionComments int
}

// defaultAmountsは実行1回分が作る件数。テストは自前の件数を渡す。
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

// stateは各生成器が作ったものを、後続の生成器へ引き渡す。実行が最初の生成器の
// 前に用意したものも合わせて運ぶ。
type state struct {
	users  *seededUsers
	spaces *seededSpaces
	topics *seededTopics
	// draftStampsは下書きを書くすべての生成器で共有する。実行1回分の下書きの
	// 並びが、それぞれが属するフェーズではなく、実行が作成した順で決まるようにするため。
	draftStamps *draftStamps
}

// generatorは実行1回分の名前付きステップ。ステップを一覧として持つことで
// 順序が1箇所にまとまり、フェーズ番号を実行時に採番できる。番号をコメントへ
// 書き込むと、実際に走る内容とずれていくため。
type generator struct {
	name string
	run  func(ctx context.Context, st *state) error
}

// Runはデータベースを空にしてシードデータを生成する。
func (r *Runner) Run(ctx context.Context) error {
	if err := EnsureDevEnv(r.cfg.Env); err != nil {
		return err
	}

	// データベースへ触れる前に名簿を読む。名簿を読めないのは設定の誤りであり、
	// 何かを削除する前に表面化させたいため。
	roster, err := loadUserRoster(rosterPath)
	if err != nil {
		return err
	}

	if err := r.db.PingContext(ctx); err != nil {
		return fmt.Errorf("データベースへのpingに失敗: %w", err)
	}

	// これから空にするデータベースと、アカウントの供給元となる名簿を報告する。
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
			return generateSoloPages(ctx, r.db, r.out, defaultAmounts, st.users, st.spaces, st.topics)
		}},
		{name: "下書きページ", run: func(ctx context.Context, st *state) error {
			return generateDraftPages(ctx, r.db, r.out, defaultAmounts, st.spaces, st.topics, st.draftStamps)
		}},
		{name: "編集提案", run: func(ctx context.Context, st *state) error {
			return generateSuggestions(ctx, r.db, r.out, defaultAmounts, st.spaces, st.topics, st.draftStamps)
		}},
		{name: "デモスペースのページ", run: func(ctx context.Context, st *state) error {
			return generateDemoPages(ctx, r.db, r.out, st.spaces, st.topics)
		}},
		// エクスポート確認用ページを末尾に置く理由は、そのトピックがトピックの
		// 仕様の末尾に置かれているのと同じである。ページ番号は生成器の実行順に配られ、
		// ブラウザ確認はURLの番号で画面を指すため、これより上に生成器を挟むと、
		// そこまでに記録した画面がすべてずれる。
		{name: "エクスポート確認用ページ", run: func(ctx context.Context, st *state) error {
			return generateExportPages(ctx, r.db, r.out, st.spaces, st.topics)
		}},
	}

	st := &state{draftStamps: newDraftStamps(startedAt)}
	for i, g := range generators {
		slog.InfoContext(ctx, fmt.Sprintf("フェーズ %d/%d: %sを生成します", i+1, len(generators), g.name))
		if err := g.run(ctx, st); err != nil {
			return fmt.Errorf("%sの生成に失敗: %w", g.name, err)
		}
	}

	// アカウントは、生成器が実際に作った場合にだけ出力する。stateの
	// フィールドは、それを埋めるステップが走るまでnilである。生成器は依存順に
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

// EnsureDevEnvは開発環境以外での実行を拒否する。
//
// シードは管理対象の行をすべて削除し、名簿が決めたパスワードでサインインできる
// アカウントを作る。資格情報のヘルパーは、そのパスワードを尋ねた相手へ出力する。
// 開発用以外のデータベースに対しては、どちらも推奨しないのではなく実行できない
// ようにする必要がある。
//
// *config.Configではなく環境名を受け取るのは、config.Loadが未設定時の既定値を
// 補う前の生のAPP_ENVに対して、コマンド側が同じ検査を適用できるようにするため。
// すべてのガードがこの1つの関数を呼ぶため、拒否の文言がずれることがなく、文言も
// シードだけでなく対象全体を名指しする形にしている。
func EnsureDevEnv(env string) error {
	if env != "dev" {
		return fmt.Errorf("開発用データを扱うコマンドは開発環境でのみ実行できます (APP_ENV=%s)", env)
	}

	return nil
}

// currentDatabaseは接続先データベースの名前を返す。
func currentDatabase(ctx context.Context, db *sql.DB) (string, error) {
	var name string
	if err := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&name); err != nil {
		return "", fmt.Errorf("接続先データベース名の取得に失敗: %w", err)
	}

	return name, nil
}

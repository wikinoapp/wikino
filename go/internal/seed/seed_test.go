package seed

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/wikinoapp/wikino/go/internal/config"
)

// sampleAccountNameは、アカウントを名指しするテキストを組み立てるテストが、
// そのアカウントの呼び名に関心を持たない場面で使う仮の表示名。名前は名簿から来る
// ため、テキストを対象とするテストが本物の名前と突き合わせられるものは無い。
const sampleAccountName = "テスト表示名"

func TestEnsureDevEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		env     string
		wantErr bool
	}{
		{name: "開発環境では実行を許可する", env: "dev", wantErr: false},
		{name: "テスト環境では実行を拒否する", env: "test", wantErr: true},
		{name: "本番環境では実行を拒否する", env: "prod", wantErr: true},
		{name: "環境が未設定のときは実行を拒否する", env: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := EnsureDevEnv(tt.env)

			if tt.wantErr && err == nil {
				t.Error("エラーを期待したがnilだった")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("エラーは期待していないが次のエラーが返った: %v", err)
			}
		})
	}
}

func TestRunRejectsNonDevEnv(t *testing.T) {
	t.Parallel()

	// Runnerには意図的にnilのデータベースを渡している。ガードは文を1つも
	// 発行する前に実行を拒否しなければならないため、ここではnil接続で足りる。
	// 仮にガードが機能しなくなっても、このテストは共有のテスト用データベースを
	// 他パッケージの足元でTRUNCATEするのではなくpanicする。
	err := NewRunner(nil, &config.Config{Env: "prod"}, io.Discard).Run(context.Background())

	if err == nil {
		t.Fatal("開発環境以外ではエラーを期待したがnilだった")
	}
}

// containsASCIILetterは、sがASCIIの英字を含むかを返す。シードが書く日本語の
// 本文はASCIIの数字や記号を保つため、「ASCIIだけか」を尋ねても、英単語とページ
// タイトルの連番を区別できない。
func containsASCIILetter(s string) bool {
	return strings.IndexFunc(s, func(r rune) bool {
		return r < utf8.RuneSelf && unicode.IsLetter(r)
	}) >= 0
}

// TestSeedBodiesWriteOneLinePerParagraphは、シードが書く本文のいずれもが段落を
// 複数行に折り返していないことを確認する。Markdownのページ本文は段落内の改行を
// <br> として描画し、編集提案の本文とコメントはwhite-space: pre-wrapによって改行を
// 保つ。手で折り返した本文は、本文が読まれる幅でブラウザが行う折り返しではなく、その
// 改行を画面に見せることになる。画面を開いて見たいのは前者のほうである。
func TestSeedBodiesWriteOneLinePerParagraph(t *testing.T) {
	t.Parallel()

	pageTitle := topicNameHandbook + " 001"

	bodies := map[string]string{
		"ピン留めページ":         pinnedPageBody(pageTitle),
		"ゴミ箱のページ":         trashedPageBody(pageTitle),
		"リンクハブ":           linkHubBody(3),
		"ハブ被リンク":          hubBacklinkBody("ハブ被リンク 01"),
		"ネスト被リンク":         nestedBacklinkBody("ネスト被リンク 01", "リンク先 01"),
		"長いタイトルのページ":      longTitlePageBody,
		"マルチバイトタイトルのページ":  multibyteTitlePageBody,
		"横に長いテーブル":        wideTableBody(),
		"長いコードブロック":       longCodeBlockBody(),
		"個人ノートのページ":       soloNotesPageBody("個人ノート 01", sampleAccountName),
		"個人シークレットのページ":    soloSecretPageBody("個人シークレット 01", sampleAccountName, sampleAccountName),
		"公開済みページの下書き":     publishedPageDraftIntro(pageTitle),
		"通常の編集提案":         ordinarySuggestionBody(pageTitle),
		"提案ページが足す節":       suggestionAddedSection,
		"Markdown記法紹介ページ": markdownGuideBody,
	}
	for _, spec := range exportPageSpecs() {
		bodies["エクスポート確認用ページ "+spec.title] = spec.body
	}
	for i, body := range bulkPageBodies {
		bodies[fmt.Sprintf("ページネーション用ページ %d", i+1)] = fmt.Sprintf(body, pageTitle)
	}
	// デモページの本文はリテラルではなくファイルであり、手で書かれて1枚ずつ
	// 足されていく。段落が誰にも気づかれずに折り返されるのはそういうときである。
	demoPages, err := loadDemoPages()
	if err != nil {
		t.Fatalf("デモページ本文の読み込みに失敗: %v", err)
	}
	for _, page := range demoPages {
		bodies["デモページ "+page.title] = page.body
	}
	for i, spec := range newPageDraftSpecs() {
		bodies[fmt.Sprintf("未公開ページの下書き %d", i+1)] = draftRevisionBody(spec.intro, 2)
	}
	for i, spec := range showcaseSuggestionSpecs() {
		bodies[fmt.Sprintf("固有の状態を見せる編集提案 %d", i+1)] = spec.body
	}
	for i, body := range suggestionCommentBodies {
		bodies[fmt.Sprintf("編集提案のコメント %d", i+1)] = fmt.Sprintf(body, i+1)
	}

	for name, body := range bodies {
		if line, number := wrappedParagraphLine(body); line != "" {
			t.Errorf("%sの本文の%d行目が段落の途中で折り返している: %q", name, number, line)
		}
	}
}

func TestWrappedParagraphLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		wantLine   string
		wantNumber int
	}{
		{
			name:       "段落内の改行を検出する",
			body:       "段落の最初の行です。\n同じ段落の次の行です。",
			wantLine:   "段落の最初の行です。",
			wantNumber: 1,
		},
		{
			name: "ハイフンの箇条書きは除外する",
			body: "- 1 つ目の項目\n- 2 つ目の項目",
		},
		{
			name: "アスタリスクの箇条書きは除外する",
			body: "* 1 つ目の項目\n* 2 つ目の項目",
		},
		{
			name: "プラスの箇条書きは除外する",
			body: "+ 1 つ目の項目\n+ 2 つ目の項目",
		},
		{
			name: "ピリオドの番号付き箇条書きは除外する",
			body: "1. 1 つ目の項目\n2. 2 つ目の項目",
		},
		{
			name: "閉じ括弧の番号付き箇条書きは除外する",
			body: "1) 1 つ目の項目\n2) 2 つ目の項目",
		},
		{
			name:       "引用内の段落改行を検出する",
			body:       "> 引用の最初の行です。\n> 引用の次の行です。",
			wantLine:   "> 引用の最初の行です。",
			wantNumber: 1,
		},
		{
			name: "コードフェンス内の行は除外する",
			body: "```text\nコードの最初の行\nコードの次の行\n```",
		},
		{
			name: "コードフェンスの中の短いフェンスでは閉じない",
			body: "````markdown\n```\nコードの最初の行\nコードの次の行\n```\n````",
		},
		{
			name: "水平線は除外する",
			body: "---\n***\n___",
		},
		{
			name: "HTMLの要素だけの行は除外する",
			body: "<details>\n<summary>クリックして展開</summary>",
		},
		{
			name:       "自動リンクはHTMLの要素として扱わない",
			body:       "<https://example.com> は自動リンクです。\n同じ段落の次の行です。",
			wantLine:   "<https://example.com> は自動リンクです。",
			wantNumber: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotLine, gotNumber := wrappedParagraphLine(tt.body)
			if gotLine != tt.wantLine || gotNumber != tt.wantNumber {
				t.Errorf(
					"wrappedParagraphLine() = (%q, %d)、期待値は (%q, %d)",
					gotLine, gotNumber, tt.wantLine, tt.wantNumber,
				)
			}
		})
	}
}

// wrappedParagraphLineは、bodyの中で次の行に続いている最初の行と、その行番号
// (1始まり) を返す。段落がすべて1行に収まっていれば空文字列を返す。対象は地の文だけ
// とする。見出し・箇条書き・表の行は記法そのものが1項目1行であり、コードフェンスの
// 中の行は段落ではなくブロックの中身であるため。
func wrappedParagraphLine(body string) (string, int) {
	lines := strings.Split(body, "\n")
	fence := 0

	for i, line := range lines {
		if backticks := codeFenceBackticks(line); backticks > 0 {
			// フェンスが閉じるのは、開いたときと同じ数以上のバックティックに
			// 限る。Markdownの記法を紹介する本文はコードブロックの中にコード
			// ブロックを見せており、内側のフェンスは外側の中身であるため。
			switch {
			case fence == 0:
				fence = backticks
			case backticks >= fence:
				fence = 0
			}

			continue
		}
		if fence > 0 || !isProseLine(line) {
			continue
		}
		if i+1 < len(lines) && isProseLine(lines[i+1]) {
			return line, i + 1
		}
	}

	return "", 0
}

// codeFenceBackticksは、lineのフェンスを開く / 閉じるバックティックの数を
// 返す。lineがフェンスでなければ0を返す。
func codeFenceBackticks(line string) int {
	trimmed := strings.TrimSpace(line)

	count := 0
	for count < len(trimmed) && trimmed[count] == '`' {
		count++
	}
	if count < 3 {
		return 0
	}

	return count
}

// isProseLineは、lineが段落の本文かどうかを返す。判定の前に引用の記号を外すのは、
// 引用された段落にも引用でない段落と同じ「1段落1行」の規則を課すため。水平線と、
// HTMLの要素だけからなる行は、地の文ではなくブロックの記法である。記法そのものが
// 1行を占め、この規則が防ごうとしている <br> にもならない。
func isProseLine(line string) bool {
	trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), ">"))
	if trimmed == "" {
		return false
	}
	if isMarkdownListItem(trimmed) || isThematicBreak(trimmed) || isHTMLElementLine(trimmed) {
		return false
	}

	for _, prefix := range []string{"#", "|", "> ", "```"} {
		if strings.HasPrefix(trimmed, prefix) {
			return false
		}
	}

	return true
}

// isThematicBreakは、lineが水平線かどうかを返す。水平線は - か * か _ の
// いずれか同じ文字を3つ以上並べた行で、他に置けるのは空白だけである。
func isThematicBreak(line string) bool {
	marker := rune(line[0])
	if marker != '-' && marker != '*' && marker != '_' {
		return false
	}

	count := 0
	for _, r := range line {
		switch r {
		case marker:
			count++
		case ' ', '\t':
		default:
			return false
		}
	}

	return count >= 3
}

// isHTMLElementLineは、lineがHTML要素の開始・終了、あるいは要素1つ分
// そのものだけであるかを返す。タグ名まで見るのは、<https://example.com> のような
// 自動リンクを記法と取り違えないため。そちらは段落として折り返されうる。
func isHTMLElementLine(line string) bool {
	if !strings.HasPrefix(line, "<") || !strings.HasSuffix(line, ">") {
		return false
	}
	if strings.HasPrefix(line, "<!--") {
		return true
	}

	name := strings.TrimPrefix(line[1:], "/")
	end := strings.IndexFunc(name, func(r rune) bool {
		return !isTagNameRune(r)
	})
	if end <= 0 {
		return false
	}

	return name[end] == '>' || name[end] == ' ' || name[end] == '/'
}

// isTagNameRuneは、rがHTMLのタグ名に現れうる文字かどうかを返す。
func isTagNameRune(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-'
}

// isMarkdownListItemは、lineがMarkdownの番号なしまたは番号付きリストの
// マーカーで始まるかを返す。番号付きマーカーは1〜9桁のASCII数字とピリオドまたは
// 閉じ括弧からなり、空でない項目ではマーカーと内容の間を空白またはタブで区切る。
func isMarkdownListItem(line string) bool {
	if line == "" {
		return false
	}

	markerEnd := 0
	switch line[0] {
	case '-', '+', '*':
		markerEnd = 1
	default:
		for markerEnd < len(line) && markerEnd < 9 && line[markerEnd] >= '0' && line[markerEnd] <= '9' {
			markerEnd++
		}
		if markerEnd == 0 || markerEnd >= len(line) || (line[markerEnd] != '.' && line[markerEnd] != ')') {
			return false
		}
		markerEnd++
	}

	return markerEnd == len(line) || line[markerEnd] == ' ' || line[markerEnd] == '\t'
}

// TestSeedTextsCarryNoRosterDisplayNameは、シードが書くテキストのいずれもが、
// 名簿の与える表示名を書き下していないことを確認する。アカウントが何と呼ばれるかは
// 設定であり、見本の名簿をコピーして自分の名前を入れた開発環境は、そうしないと、
// そこには存在しないアカウントを名指しするページやトピック一覧を開くことになる。
// しかもそのことは何からも告げられない。
//
// 名前を見本の名簿から取るのは、名簿がそこからコピーされるファイルであり、それに
// よって、テキストが書かれる際に照らされる名前がそこの名前になるため。
//
// テキストを1つずつ並べるのではなくパッケージ自身のソースを読むのは、後から足した
// テキストが一覧にも足される必要が出るため。ここで確認しているのは、次に何が書かれても
// 適用される規則である。
func TestSeedTextsCarryNoRosterDisplayName(t *testing.T) {
	t.Parallel()

	names := exampleRosterDisplayNames(t)

	for _, text := range seedTexts(t) {
		for role, name := range names {
			if strings.Contains(text.text, name) {
				t.Errorf("%sが名簿の表示名%q (%s) を直に書いている", text.file, name, role)
			}
		}
	}
}

// exampleRosterDisplayNamesは、見本の名簿が挙げるアカウントの表示名を、その
// アカウントが作成される役割をキーにして返す。
func exampleRosterDisplayNames(t *testing.T) map[seedRole]string {
	t.Helper()

	file, err := loadRosterFile(filepath.Join("..", "..", rosterExamplePath))
	if err != nil {
		t.Fatalf("%sの読み込みに失敗: %v", rosterExamplePath, err)
	}
	users, err := file.validate()
	if err != nil {
		t.Fatalf("%sの検査に失敗: %v", rosterExamplePath, err)
	}

	names := make(map[seedRole]string, len(users))
	for _, user := range users {
		names[user.role] = user.name
	}

	return names
}

// seedTextはパッケージが持つ文字列1つと、それが書かれているファイル。
type seedText struct {
	file string
	text string
}

// seedTextsは、シードが画面へ書きうる文字列を返す。パッケージ自身のソースの
// 文字列リテラルすべてと、ファイルとして埋め込んでいる本文である。
//
// テストファイルは対象から外す。テストがアカウントを名指しするのは意図的であり、
// それが「そのテキストがどのアカウントを名指しするはずか」を述べる方法になっている。
// テストにも同じ規則を課すと、その期待値を書き表す方法が無くなる。
func seedTexts(t *testing.T) []seedText {
	t.Helper()

	// テストバイナリはパッケージのディレクトリを作業ディレクトリとして動くため、
	// ソースも埋め込む本文も同じ場所にある。parser.ParseDirではなくファイルを列挙して
	// 1つずつ解析するのは、同関数がビルドタグを無視するとして非推奨になっているため。
	// ここではタグに関わらずすべての .goファイルが対象でよい。
	const (
		dir       = "."
		bodiesDir = "bodies"
	)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("%sの読み取りに失敗: %v", dir, err)
	}

	fset := token.NewFileSet()
	texts := make([]seedText, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			t.Fatalf("%sの解析に失敗: %v", name, err)
		}

		ast.Inspect(file, func(node ast.Node) bool {
			lit, ok := node.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}

			// リテラルの引用符を外すのは、エスケープシーケンスを、それが表す
			// 文字として比較するため。画面に出るのはそちらであるため。
			value, err := strconv.Unquote(lit.Value)
			if err != nil {
				t.Errorf("%sの文字列リテラル%sを読み取れない: %v", name, lit.Value, err)

				return true
			}
			texts = append(texts, seedText{file: name, text: value})

			return true
		})
	}

	// 本文は列挙ではなく走査で集める。go:embedはサブディレクトリの中まで
	// 届くため。サブディレクトリへ置いた本文は、そうしないと規則から黙って外れる。
	// ソースを読む形にしたのは、それを避けるためであった。
	if err := filepath.WalkDir(bodiesDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		texts = append(texts, seedText{file: path, text: string(content)})

		return nil
	}); err != nil {
		t.Fatalf("%sの読み取りに失敗: %v", bodiesDir, err)
	}

	return texts
}

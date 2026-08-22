package seed

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/wikinoapp/wikino/go/internal/config"
)

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

	// The Runner is given a nil database on purpose. The guard has to reject
	// the run before any statement is issued, so a nil connection is enough
	// here — and if the guard ever stopped working, this test would panic
	// instead of truncating the shared test database out from under the other
	// packages.
	//
	// [Ja] Runner には意図的に nil のデータベースを渡している。ガードは文を 1 つも
	// 発行する前に実行を拒否しなければならないため、ここでは nil 接続で足りる。
	// 仮にガードが機能しなくなっても、このテストは共有のテスト用データベースを
	// 他パッケージの足元で TRUNCATE するのではなく panic する。
	err := NewRunner(nil, &config.Config{Env: "prod"}, io.Discard).Run(context.Background())

	if err == nil {
		t.Fatal("開発環境以外ではエラーを期待したがnilだった")
	}
}

// containsASCIILetter reports whether s holds a letter of the ASCII alphabet.
// The Japanese copy the seed writes keeps ASCII digits and punctuation, so
// asking whether a body is ASCII-only would not tell an English word apart from
// the number on a page title.
//
// [Ja] containsASCIILetter は、s が ASCII の英字を含むかを返す。シードが書く日本語の
// 本文は ASCII の数字や記号を保つため、「ASCII だけか」を尋ねても、英単語とページ
// タイトルの連番を区別できない。
func containsASCIILetter(s string) bool {
	return strings.IndexFunc(s, func(r rune) bool {
		return r < utf8.RuneSelf && unicode.IsLetter(r)
	}) >= 0
}

// TestSeedBodiesWriteOneLinePerParagraph checks that no body the seed writes
// wraps a paragraph across lines. Markdown page bodies render a line break
// inside a paragraph as a <br>, while suggestion bodies and comments preserve
// it through white-space: pre-wrap. A body wrapped by hand therefore shows those
// breaks on the screen instead of the wrapping the browser does at the width the
// body is read in, which is what these screens are opened to look at.
//
// The Markdown guide is not among the bodies checked here: it is a Markdown
// file rather than a Go literal, and the shared Markdown linter asks those for
// one sentence per line.
//
// [Ja] TestSeedBodiesWriteOneLinePerParagraph は、シードが書く本文のいずれもが段落を
// 複数行に折り返していないことを確認する。Markdown のページ本文は段落内の改行を
// <br> として描画し、編集提案の本文とコメントは white-space: pre-wrap によって改行を
// 保つ。手で折り返した本文は、本文が読まれる幅でブラウザが行う折り返しではなく、その
// 改行を画面に見せることになる。画面を開いて見たいのは前者のほうである。
//
// Markdown 記法紹介ページはここで確認する本文に含めない。Go のリテラルではなく Markdown
// ファイルであり、共通の Markdown リンタが句点改行を求めるため。
func TestSeedBodiesWriteOneLinePerParagraph(t *testing.T) {
	t.Parallel()

	pageTitle := topicNameHandbook + " 001"

	bodies := map[string]string{
		"ピン留めページ":        pinnedPageBody(pageTitle),
		"ゴミ箱のページ":        trashedPageBody(pageTitle),
		"リンクハブ":          linkHubBody(3),
		"ハブ被リンク":         hubBacklinkBody("ハブ被リンク 01"),
		"ネスト被リンク":        nestedBacklinkBody("ネスト被リンク 01", "リンク先 01"),
		"長いタイトルのページ":     longTitlePageBody,
		"マルチバイトタイトルのページ": multibyteTitlePageBody,
		"横に長いテーブル":       wideTableBody(),
		"長いコードブロック":      longCodeBlockBody(),
		"個人ノートのページ":      soloNotesPageBody("個人ノート 01"),
		"個人シークレットのページ":   soloSecretPageBody("個人シークレット 01"),
		"公開済みページの下書き":    publishedPageDraftIntro(pageTitle),
		"通常の編集提案":        ordinarySuggestionBody(pageTitle),
		"提案ページが足す節":      suggestionAddedSection,
	}
	for i, body := range bulkPageBodies {
		bodies[fmt.Sprintf("ページネーション用ページ %d", i+1)] = fmt.Sprintf(body, pageTitle)
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
			t.Errorf("%s の本文の %d 行目が段落の途中で折り返している: %q", name, number, line)
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

// wrappedParagraphLine returns the first line of body that another line
// continues, together with its 1-based number, or an empty string when every
// paragraph sits on one line. Only prose is looked at: headings, list items and
// table rows are one line per item by their own syntax, and the lines inside a
// fenced code block are the block's content rather than a paragraph.
//
// [Ja] wrappedParagraphLine は、body の中で次の行に続いている最初の行と、その行番号
// (1 始まり) を返す。段落がすべて 1 行に収まっていれば空文字列を返す。対象は地の文だけ
// とする。見出し・箇条書き・表の行は記法そのものが 1 項目 1 行であり、コードフェンスの
// 中の行は段落ではなくブロックの中身であるため。
func wrappedParagraphLine(body string) (string, int) {
	lines := strings.Split(body, "\n")
	inFence := false

	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if inFence || !isProseLine(line) {
			continue
		}
		if i+1 < len(lines) && isProseLine(lines[i+1]) {
			return line, i + 1
		}
	}

	return "", 0
}

// isProseLine reports whether line carries paragraph text. A blockquote marker
// is stripped before the line is judged, so that a quoted paragraph is held to
// the same one-line rule as an unquoted one.
//
// [Ja] isProseLine は、line が段落の本文かどうかを返す。判定の前に引用の記号を外すのは、
// 引用された段落にも引用でない段落と同じ「1 段落 1 行」の規則を課すため。
func isProseLine(line string) bool {
	trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), ">"))
	if trimmed == "" {
		return false
	}
	if isMarkdownListItem(trimmed) {
		return false
	}

	for _, prefix := range []string{"#", "|", "> ", "```"} {
		if strings.HasPrefix(trimmed, prefix) {
			return false
		}
	}

	return true
}

// isMarkdownListItem reports whether line starts with an unordered or ordered
// Markdown list marker. An ordered marker has one to nine ASCII digits followed
// by a period or closing parenthesis, and a nonempty item separates its marker
// from its content with a space or tab.
//
// [Ja] isMarkdownListItem は、line が Markdown の番号なしまたは番号付きリストの
// マーカーで始まるかを返す。番号付きマーカーは 1〜9 桁の ASCII 数字とピリオドまたは
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

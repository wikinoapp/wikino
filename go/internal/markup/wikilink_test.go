package markup

import (
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestScanWikilinks_SinglePageName(t *testing.T) {
	t.Parallel()

	keys := ScanWikilinks("テキスト [[ページ1]] テキスト", "トピックA")

	if len(keys) != 1 {
		t.Fatalf("len(keys) = %d, want 1", len(keys))
	}
	if keys[0].Raw != "ページ1" {
		t.Errorf("Raw = %q, want %q", keys[0].Raw, "ページ1")
	}
	if keys[0].TopicName != "トピックA" {
		t.Errorf("TopicName = %q, want %q", keys[0].TopicName, "トピックA")
	}
	if keys[0].PageTitle != "ページ1" {
		t.Errorf("PageTitle = %q, want %q", keys[0].PageTitle, "ページ1")
	}
}

func TestScanWikilinks_TopicAndPageName(t *testing.T) {
	t.Parallel()

	keys := ScanWikilinks("テキスト [[トピックB/ページ2]] テキスト", "トピックA")

	if len(keys) != 1 {
		t.Fatalf("len(keys) = %d, want 1", len(keys))
	}
	if keys[0].Raw != "トピックB/ページ2" {
		t.Errorf("Raw = %q, want %q", keys[0].Raw, "トピックB/ページ2")
	}
	if keys[0].TopicName != "トピックB" {
		t.Errorf("TopicName = %q, want %q", keys[0].TopicName, "トピックB")
	}
	if keys[0].PageTitle != "ページ2" {
		t.Errorf("PageTitle = %q, want %q", keys[0].PageTitle, "ページ2")
	}
}

func TestScanWikilinks_MultipleWikilinks(t *testing.T) {
	t.Parallel()

	keys := ScanWikilinks("[[ページ1]] テキスト [[トピックB/ページ2]] テキスト [[ページ3]]", "トピックA")

	if len(keys) != 3 {
		t.Fatalf("len(keys) = %d, want 3", len(keys))
	}

	// 1つ目: ページ名のみ
	if keys[0].TopicName != "トピックA" || keys[0].PageTitle != "ページ1" {
		t.Errorf("keys[0] = {TopicName: %q, PageTitle: %q}, want {トピックA, ページ1}", keys[0].TopicName, keys[0].PageTitle)
	}

	// 2つ目: トピック名/ページ名
	if keys[1].TopicName != "トピックB" || keys[1].PageTitle != "ページ2" {
		t.Errorf("keys[1] = {TopicName: %q, PageTitle: %q}, want {トピックB, ページ2}", keys[1].TopicName, keys[1].PageTitle)
	}

	// 3つ目: ページ名のみ
	if keys[2].TopicName != "トピックA" || keys[2].PageTitle != "ページ3" {
		t.Errorf("keys[2] = {TopicName: %q, PageTitle: %q}, want {トピックA, ページ3}", keys[2].TopicName, keys[2].PageTitle)
	}
}

func TestScanWikilinks_EmptyBrackets(t *testing.T) {
	t.Parallel()

	keys := ScanWikilinks("テキスト [[]] テキスト", "トピックA")

	if len(keys) != 0 {
		t.Errorf("len(keys) = %d, want 0 (empty brackets should be skipped)", len(keys))
	}
}

func TestScanWikilinks_WhitespaceOnly(t *testing.T) {
	t.Parallel()

	keys := ScanWikilinks("テキスト [[  ]] テキスト", "トピックA")

	if len(keys) != 0 {
		t.Errorf("len(keys) = %d, want 0 (whitespace-only should be skipped)", len(keys))
	}
}

func TestScanWikilinks_NoWikilinks(t *testing.T) {
	t.Parallel()

	keys := ScanWikilinks("Wikiリンクなしのテキスト", "トピックA")

	if keys != nil {
		t.Errorf("keys = %v, want nil", keys)
	}
}

func TestScanWikilinks_EmptyBody(t *testing.T) {
	t.Parallel()

	keys := ScanWikilinks("", "トピックA")

	if keys != nil {
		t.Errorf("keys = %v, want nil", keys)
	}
}

func TestScanWikilinks_SlashInPageTitle(t *testing.T) {
	t.Parallel()

	// /で最大2分割なので、2つ目以降の/はページタイトルの一部
	keys := ScanWikilinks("[[トピックA/ページ/サブページ]]", "デフォルト")

	if len(keys) != 1 {
		t.Fatalf("len(keys) = %d, want 1", len(keys))
	}
	if keys[0].TopicName != "トピックA" {
		t.Errorf("TopicName = %q, want %q", keys[0].TopicName, "トピックA")
	}
	if keys[0].PageTitle != "ページ/サブページ" {
		t.Errorf("PageTitle = %q, want %q", keys[0].PageTitle, "ページ/サブページ")
	}
}

func TestScanWikilinks_TrimWhitespace(t *testing.T) {
	t.Parallel()

	keys := ScanWikilinks("[[ ページ1 ]]", "トピックA")

	if len(keys) != 1 {
		t.Fatalf("len(keys) = %d, want 1", len(keys))
	}
	if keys[0].Raw != "ページ1" {
		t.Errorf("Raw = %q, want %q", keys[0].Raw, "ページ1")
	}
	if keys[0].PageTitle != "ページ1" {
		t.Errorf("PageTitle = %q, want %q", keys[0].PageTitle, "ページ1")
	}
}

func TestScanWikilinks_SpecialCharacters(t *testing.T) {
	t.Parallel()

	keys := ScanWikilinks("[[日記 (2025)]]", "トピックA")

	if len(keys) != 1 {
		t.Fatalf("len(keys) = %d, want 1", len(keys))
	}
	if keys[0].PageTitle != "日記 (2025)" {
		t.Errorf("PageTitle = %q, want %q", keys[0].PageTitle, "日記 (2025)")
	}
}

func TestReplaceWikilinks_ExistingPage(t *testing.T) {
	t.Parallel()

	bodyHTML := "<p>テキスト [[ページ1]] テキスト</p>"
	locations := []PageLocation{
		{
			Key:        WikilinkKey{Raw: "ページ1", TopicName: "トピックA", PageTitle: "ページ1"},
			TopicName:  "トピックA",
			PageID:     model.PageID("page-id-1"),
			PageNumber: 42,
			PageTitle:  "ページ1",
		},
	}

	got := ReplaceWikilinks(bodyHTML, "トピックA", "my-space", locations)
	want := `<p>テキスト <a href="/s/my-space/pages/42">ページ1</a> テキスト</p>`

	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReplaceWikilinks_NonExistingPage(t *testing.T) {
	t.Parallel()

	bodyHTML := "<p>テキスト [[存在しないページ]] テキスト</p>"
	var locations []PageLocation

	got := ReplaceWikilinks(bodyHTML, "トピックA", "my-space", locations)

	// 存在しないページはプレーンテキストのまま残す
	if got != bodyHTML {
		t.Errorf("got:\n%s\nwant:\n%s", got, bodyHTML)
	}
}

func TestReplaceWikilinks_TopicAndPageName(t *testing.T) {
	t.Parallel()

	bodyHTML := "<p>[[トピックB/ページ2]]</p>"
	locations := []PageLocation{
		{
			Key:        WikilinkKey{Raw: "トピックB/ページ2", TopicName: "トピックB", PageTitle: "ページ2"},
			TopicName:  "トピックB",
			PageID:     model.PageID("page-id-2"),
			PageNumber: 99,
			PageTitle:  "ページ2",
		},
	}

	got := ReplaceWikilinks(bodyHTML, "トピックA", "my-space", locations)
	want := `<p><a href="/s/my-space/pages/99">ページ2</a></p>`

	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReplaceWikilinks_MultipleLinks(t *testing.T) {
	t.Parallel()

	bodyHTML := "<p>[[ページ1]] と [[トピックB/ページ2]] と [[存在しない]]</p>"
	locations := []PageLocation{
		{
			Key:        WikilinkKey{Raw: "ページ1", TopicName: "トピックA", PageTitle: "ページ1"},
			TopicName:  "トピックA",
			PageID:     model.PageID("page-id-1"),
			PageNumber: 1,
			PageTitle:  "ページ1",
		},
		{
			Key:        WikilinkKey{Raw: "トピックB/ページ2", TopicName: "トピックB", PageTitle: "ページ2"},
			TopicName:  "トピックB",
			PageID:     model.PageID("page-id-2"),
			PageNumber: 2,
			PageTitle:  "ページ2",
		},
	}

	got := ReplaceWikilinks(bodyHTML, "トピックA", "my-space", locations)
	want := `<p><a href="/s/my-space/pages/1">ページ1</a> と <a href="/s/my-space/pages/2">ページ2</a> と [[存在しない]]</p>`

	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReplaceWikilinks_SkipInsideCodeTag(t *testing.T) {
	t.Parallel()

	bodyHTML := "<p>テキスト</p><code>[[ページ1]]</code><p>テキスト</p>"
	locations := []PageLocation{
		{
			Key:        WikilinkKey{Raw: "ページ1", TopicName: "トピックA", PageTitle: "ページ1"},
			TopicName:  "トピックA",
			PageID:     model.PageID("page-id-1"),
			PageNumber: 1,
			PageTitle:  "ページ1",
		},
	}

	got := ReplaceWikilinks(bodyHTML, "トピックA", "my-space", locations)

	// <code>内のWikiリンクは変換されない
	if got != bodyHTML {
		t.Errorf("got:\n%s\nwant:\n%s (should not replace inside <code>)", got, bodyHTML)
	}
}

func TestReplaceWikilinks_SkipInsidePreTag(t *testing.T) {
	t.Parallel()

	bodyHTML := "<pre>[[ページ1]]</pre>"
	locations := []PageLocation{
		{
			Key:        WikilinkKey{Raw: "ページ1", TopicName: "トピックA", PageTitle: "ページ1"},
			TopicName:  "トピックA",
			PageID:     model.PageID("page-id-1"),
			PageNumber: 1,
			PageTitle:  "ページ1",
		},
	}

	got := ReplaceWikilinks(bodyHTML, "トピックA", "my-space", locations)

	if got != bodyHTML {
		t.Errorf("got:\n%s\nwant:\n%s (should not replace inside <pre>)", got, bodyHTML)
	}
}

func TestReplaceWikilinks_SkipInsideATag(t *testing.T) {
	t.Parallel()

	bodyHTML := `<a href="/example">[[ページ1]]</a>`
	locations := []PageLocation{
		{
			Key:        WikilinkKey{Raw: "ページ1", TopicName: "トピックA", PageTitle: "ページ1"},
			TopicName:  "トピックA",
			PageID:     model.PageID("page-id-1"),
			PageNumber: 1,
			PageTitle:  "ページ1",
		},
	}

	got := ReplaceWikilinks(bodyHTML, "トピックA", "my-space", locations)

	if got != bodyHTML {
		t.Errorf("got:\n%s\nwant:\n%s (should not replace inside <a>)", got, bodyHTML)
	}
}

func TestReplaceWikilinks_SkipInsideScriptTag(t *testing.T) {
	t.Parallel()

	bodyHTML := "<script>var x = '[[ページ1]]';</script>"
	locations := []PageLocation{
		{
			Key:        WikilinkKey{Raw: "ページ1", TopicName: "トピックA", PageTitle: "ページ1"},
			TopicName:  "トピックA",
			PageID:     model.PageID("page-id-1"),
			PageNumber: 1,
			PageTitle:  "ページ1",
		},
	}

	got := ReplaceWikilinks(bodyHTML, "トピックA", "my-space", locations)

	if got != bodyHTML {
		t.Errorf("got:\n%s\nwant:\n%s (should not replace inside <script>)", got, bodyHTML)
	}
}

func TestReplaceWikilinks_SkipInsideStyleTag(t *testing.T) {
	t.Parallel()

	bodyHTML := "<style>/* [[ページ1]] */</style>"
	locations := []PageLocation{
		{
			Key:        WikilinkKey{Raw: "ページ1", TopicName: "トピックA", PageTitle: "ページ1"},
			TopicName:  "トピックA",
			PageID:     model.PageID("page-id-1"),
			PageNumber: 1,
			PageTitle:  "ページ1",
		},
	}

	got := ReplaceWikilinks(bodyHTML, "トピックA", "my-space", locations)

	if got != bodyHTML {
		t.Errorf("got:\n%s\nwant:\n%s (should not replace inside <style>)", got, bodyHTML)
	}
}

func TestReplaceWikilinks_MixedSkipAndReplace(t *testing.T) {
	t.Parallel()

	bodyHTML := "<p>[[ページ1]]</p><code>[[ページ2]]</code><p>[[ページ3]]</p>"
	locations := []PageLocation{
		{
			Key:        WikilinkKey{Raw: "ページ1", TopicName: "トピックA", PageTitle: "ページ1"},
			TopicName:  "トピックA",
			PageID:     model.PageID("page-id-1"),
			PageNumber: 1,
			PageTitle:  "ページ1",
		},
		{
			Key:        WikilinkKey{Raw: "ページ2", TopicName: "トピックA", PageTitle: "ページ2"},
			TopicName:  "トピックA",
			PageID:     model.PageID("page-id-2"),
			PageNumber: 2,
			PageTitle:  "ページ2",
		},
		{
			Key:        WikilinkKey{Raw: "ページ3", TopicName: "トピックA", PageTitle: "ページ3"},
			TopicName:  "トピックA",
			PageID:     model.PageID("page-id-3"),
			PageNumber: 3,
			PageTitle:  "ページ3",
		},
	}

	got := ReplaceWikilinks(bodyHTML, "トピックA", "my-space", locations)
	want := `<p><a href="/s/my-space/pages/1">ページ1</a></p><code>[[ページ2]]</code><p><a href="/s/my-space/pages/3">ページ3</a></p>`

	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReplaceWikilinks_NoWikilinks(t *testing.T) {
	t.Parallel()

	bodyHTML := "<p>Wikiリンクなしのテキスト</p>"

	got := ReplaceWikilinks(bodyHTML, "トピックA", "my-space", nil)

	if got != bodyHTML {
		t.Errorf("got:\n%s\nwant:\n%s", got, bodyHTML)
	}
}

func TestReplaceWikilinks_EmptyInput(t *testing.T) {
	t.Parallel()

	got := ReplaceWikilinks("", "トピックA", "my-space", nil)

	if got != "" {
		t.Errorf("got: %q, want empty string", got)
	}
}

func TestReplaceWikilinks_HTMLEscaping(t *testing.T) {
	t.Parallel()

	// DOMパーサーは<script>を実際のscript要素として解釈するため、
	// テスト入力にはHTMLエスケープ済み文字列を使用する
	bodyHTML := `<p>[[ページ&lt;script&gt;]]</p>`
	locations := []PageLocation{
		{
			Key:        WikilinkKey{Raw: "ページ<script>", TopicName: "トピックA", PageTitle: "ページ<script>"},
			TopicName:  "トピックA",
			PageID:     model.PageID("page-id-1"),
			PageNumber: 1,
			PageTitle:  "ページ<script>",
		},
	}

	got := ReplaceWikilinks(bodyHTML, "トピックA", "my-space", locations)

	// ページタイトルのHTMLエスケープを確認
	want := `<p><a href="/s/my-space/pages/1">ページ&lt;script&gt;</a></p>`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReplaceWikilinks_SpecialCharacters(t *testing.T) {
	t.Parallel()

	bodyHTML := "<p>[[日記 (2025)]]</p>"
	locations := []PageLocation{
		{
			Key:        WikilinkKey{Raw: "日記 (2025)", TopicName: "トピックA", PageTitle: "日記 (2025)"},
			TopicName:  "トピックA",
			PageID:     model.PageID("page-id-1"),
			PageNumber: 5,
			PageTitle:  "日記 (2025)",
		},
	}

	got := ReplaceWikilinks(bodyHTML, "トピックA", "my-space", locations)
	want := `<p><a href="/s/my-space/pages/5">日記 (2025)</a></p>`

	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReplaceWikilinks_SkipInsideNestedCodeInPre(t *testing.T) {
	t.Parallel()

	bodyHTML := "<pre><code>[[ページ1]]</code></pre>"
	locations := []PageLocation{
		{
			Key:        WikilinkKey{Raw: "ページ1", TopicName: "トピックA", PageTitle: "ページ1"},
			TopicName:  "トピックA",
			PageID:     model.PageID("page-id-1"),
			PageNumber: 1,
			PageTitle:  "ページ1",
		},
	}

	got := ReplaceWikilinks(bodyHTML, "トピックA", "my-space", locations)

	if got != bodyHTML {
		t.Errorf("got:\n%s\nwant:\n%s (should not replace inside <pre><code>)", got, bodyHTML)
	}
}

func TestScanWikilinks_TripleBrackets(t *testing.T) {
	t.Parallel()

	// Obsidian互換: [[[a]]] は [[[a]] として解釈され、rawは "[a" になる
	keys := ScanWikilinks("[[[a]]]", "トピックA")

	if len(keys) != 1 {
		t.Fatalf("len(keys) = %d, want 1", len(keys))
	}
	if keys[0].Raw != "[a" {
		t.Errorf("Raw = %q, want %q", keys[0].Raw, "[a")
	}
	if keys[0].TopicName != "トピックA" {
		t.Errorf("TopicName = %q, want %q", keys[0].TopicName, "トピックA")
	}
	if keys[0].PageTitle != "[a" {
		t.Errorf("PageTitle = %q, want %q", keys[0].PageTitle, "[a")
	}
}

func TestScanWikilinks_TripleBracketsWithSpaces(t *testing.T) {
	t.Parallel()

	// Obsidian互換: [[[ a ]]] は rawが "[ a" になる
	keys := ScanWikilinks("[[[ a ]]]", "トピックA")

	if len(keys) != 1 {
		t.Fatalf("len(keys) = %d, want 1", len(keys))
	}
	if keys[0].Raw != "[ a" {
		t.Errorf("Raw = %q, want %q", keys[0].Raw, "[ a")
	}
	if keys[0].PageTitle != "[ a" {
		t.Errorf("PageTitle = %q, want %q", keys[0].PageTitle, "[ a")
	}
}

func TestScanWikilinks_TripleBracketsMixed(t *testing.T) {
	t.Parallel()

	// [[[a]]] と [[b]] の混在
	keys := ScanWikilinks("[[[a]]] [[b]]", "トピックA")

	if len(keys) != 2 {
		t.Fatalf("len(keys) = %d, want 2", len(keys))
	}
	if keys[0].Raw != "[a" {
		t.Errorf("keys[0].Raw = %q, want %q", keys[0].Raw, "[a")
	}
	if keys[1].Raw != "b" {
		t.Errorf("keys[1].Raw = %q, want %q", keys[1].Raw, "b")
	}
}

func TestReplaceWikilinks_HTMLEntityAmpersand(t *testing.T) {
	t.Parallel()

	// Markdownの [[A & B]] はHTML化後に [[A &amp; B]] となる。
	// HTMLパーサーがテキストノードを自動デコードするため、
	// テキストノードのDataは "[[A & B]]" となりマッチする。
	bodyHTML := `<p>[[A &amp; B]]</p>`
	locations := []PageLocation{
		{
			Key:        WikilinkKey{Raw: "A & B", TopicName: "トピックA", PageTitle: "A & B"},
			TopicName:  "トピックA",
			PageID:     model.PageID("page-id-1"),
			PageNumber: 10,
			PageTitle:  "A & B",
		},
	}

	got := ReplaceWikilinks(bodyHTML, "トピックA", "my-space", locations)
	want := `<p><a href="/s/my-space/pages/10">A &amp; B</a></p>`

	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReplaceWikilinks_HTMLEntityQuot(t *testing.T) {
	t.Parallel()

	// &quot; を含むWikiリンクのテスト
	bodyHTML := `<p>[[ページ&quot;名前&quot;]]</p>`
	locations := []PageLocation{
		{
			Key:        WikilinkKey{Raw: `ページ"名前"`, TopicName: "トピックA", PageTitle: `ページ"名前"`},
			TopicName:  "トピックA",
			PageID:     model.PageID("page-id-1"),
			PageNumber: 11,
			PageTitle:  `ページ"名前"`,
		},
	}

	got := ReplaceWikilinks(bodyHTML, "トピックA", "my-space", locations)
	want := `<p><a href="/s/my-space/pages/11">ページ&#34;名前&#34;</a></p>`

	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReplaceWikilinks_HTMLNumericEntityBrackets(t *testing.T) {
	t.Parallel()

	// 数値文字参照 &#91; ([) と &#93; (]) でブラケットがエンコードされたケース。
	// ReplaceWikilinksは最初に bodyHTML に "[[" が含まれるかチェックするため、
	// ブラケットが数値文字参照の場合は変換されない。
	// 実際のMarkdownレンダラーはブラケットをエンティティエンコードしないため、
	// このケースは実運用では発生しない。
	bodyHTML := `<p>&#91;&#91;ページ名&#93;&#93;</p>`
	locations := []PageLocation{
		{
			Key:        WikilinkKey{Raw: "ページ名", TopicName: "トピックA", PageTitle: "ページ名"},
			TopicName:  "トピックA",
			PageID:     model.PageID("page-id-1"),
			PageNumber: 12,
			PageTitle:  "ページ名",
		},
	}

	got := ReplaceWikilinks(bodyHTML, "トピックA", "my-space", locations)

	// ブラケットがエンティティエンコードされている場合は変換されない
	if got != bodyHTML {
		t.Errorf("got:\n%s\nwant:\n%s (should not replace when brackets are entity-encoded)", got, bodyHTML)
	}
}

func TestReplaceWikilinks_HTMLEntityArrow(t *testing.T) {
	t.Parallel()

	// Rails版のテストにある "->" 等の特殊文字を含むWikiリンク。
	// Markdownレンダラーが "&gt;" にエンコードするケースを検証する。
	bodyHTML := `<p>[[A -&gt; B]]</p>`
	locations := []PageLocation{
		{
			Key:        WikilinkKey{Raw: "A -> B", TopicName: "トピックA", PageTitle: "A -> B"},
			TopicName:  "トピックA",
			PageID:     model.PageID("page-id-1"),
			PageNumber: 13,
			PageTitle:  "A -> B",
		},
	}

	got := ReplaceWikilinks(bodyHTML, "トピックA", "my-space", locations)
	want := `<p><a href="/s/my-space/pages/13">A -&gt; B</a></p>`

	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReplaceWikilinks_SpaceIdentifierEscaping(t *testing.T) {
	t.Parallel()

	bodyHTML := "<p>[[ページ1]]</p>"
	locations := []PageLocation{
		{
			Key:        WikilinkKey{Raw: "ページ1", TopicName: "トピックA", PageTitle: "ページ1"},
			TopicName:  "トピックA",
			PageID:     model.PageID("page-id-1"),
			PageNumber: 1,
			PageTitle:  "ページ1",
		},
	}

	got := ReplaceWikilinks(bodyHTML, "トピックA", "space&id", locations)
	want := `<p><a href="/s/space&amp;id/pages/1">ページ1</a></p>`

	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestScanWikilinkMatches_ParsedLabelSyntax(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		label string
	}{
		{name: "HTML 属性の閉じ角括弧", label: `<span title="]">label</span> [[A]]`},
		{name: "HTML 属性の開き角括弧", label: `<span title="[">label</span> [[A]]`},
		{name: "HTML コメントの閉じ角括弧", label: `<!-- ] -->label [[A]]`},
		{name: "HTML コメントの開き角括弧", label: `<!-- [ -->label [[A]]`},
		{name: "子画像のリンク先の閉じ角括弧", label: `![image](https://example.com/image]x.png) [[A]]`},
		{name: "子画像のリンク先の開き角括弧", label: `![image](https://example.com/image[x.png) [[A]]`},
		{name: "子画像のタイトルの閉じ角括弧", label: `![image](https://example.com/image.png "]") [[A]]`},
		{name: "子画像のタイトルの開き角括弧", label: `![image](https://example.com/image.png "[") [[A]]`},
		{name: "子画像の中の HTML", label: `![<span title="]">image</span>](https://example.com/image.png) [[A]]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			body := "[" + tt.label + "](https://example.com/) [[B]]"
			matches := ScanWikilinkMatches(body, "T")
			if len(matches) != 1 {
				t.Fatalf("matches = %+v, want only [[B]]", matches)
			}
			match := matches[0]
			if body[match.Start:match.Stop] != "[[B]]" || match.Key.TopicName != "T" || match.Key.PageTitle != "B" {
				t.Errorf("match = %+v, want [[B]] in topic T", match)
			}
		})
	}
}

func TestScanWikilinkMatches(t *testing.T) {
	t.Parallel()

	type want struct {
		text      string
		topicName string
		pageTitle string
	}

	tests := []struct {
		name             string
		body             string
		currentTopicName string
		want             []want
	}{
		{
			name:             "トピック名付きのリンクを位置とともに返す",
			body:             "詳細は [[トピックB/ページ2]] を参照。",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[トピックB/ページ2]]", topicName: "トピックB", pageTitle: "ページ2"},
			},
		},
		{
			name:             "省略形のリンクには現在のトピック名を補う",
			body:             "詳細は [[ページ1]] を参照。",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ1]]", topicName: "トピックA", pageTitle: "ページ1"},
			},
		},
		{
			name:             "複数のリンクを現れる順に返す",
			body:             "[[ページ1]] と [[トピックB/ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ1]]", topicName: "トピックA", pageTitle: "ページ1"},
				{text: "[[トピックB/ページ2]]", topicName: "トピックB", pageTitle: "ページ2"},
			},
		},
		{
			name:             "フェンス付きコードブロックの中のリンクは返さない",
			body:             "```\n[[ページ1]]\n```\n\n[[ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			name:             "フェンス付きコードブロックの情報文字列にあるリンクは返さない",
			body:             "```[[ページ1]]\n本文\n```\n\n[[ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			name:             "インラインコードの中のリンクは返さない",
			body:             "`[[ページ1]]` と [[ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			name:             "字下げされたコードブロックの中のリンクは返さない",
			body:             "段落。\n\n    [[ページ1]]\n\n[[ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			name:             "表の中のインラインコードも対象外にする",
			body:             "| a | b |\n| --- | --- |\n| `[[ページ1]]` | [[ページ2]] |",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			name:             "raw HTML の code 要素の中のリンクは返さない",
			body:             "<code>[[ページ1]]</code> と [[ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			name:             "raw HTML の pre 要素の中のリンクは返さない",
			body:             "<pre>\n[[ページ1]]\n</pre>\n\n[[ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			name:             "自己終了記法の code 要素の後ろにあるリンクは返さない",
			body:             "<code/>[[ページ1]]</code> と [[ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			name:             "自己終了記法の pre 要素の後ろにあるリンクは返さない",
			body:             "<pre/>[[ページ1]]</pre>\n\n[[ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			// サニタイズが中身ごと落とすため、これらの要素の中に書いたリンクは画面に出ない
			name: "サニタイズが中身ごと落とす要素の中のリンクは返さない",
			body: "<iframe>[[ページ1]]</iframe>\n\n<noscript>[[ページ2]]</noscript>\n\n" +
				"<object>[[ページ3]]</object>\n\n[[ページ4]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ4]]", topicName: "トピックA", pageTitle: "ページ4"},
			},
		},
		{
			name:             "破棄要素を異なる破棄要素の終了タグで閉じた後のリンクは返す",
			body:             "<object>[[ページ1]]</iframe>[[ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			name:             "破棄要素の中の script 要素の終了タグは破棄を終わらせない",
			body:             "<object>[[ページ1]]</script>[[ページ2]]",
			currentTopicName: "トピックA",
			want:             nil,
		},
		{
			name:             "破棄要素の中の style 要素の終了タグは破棄を終わらせない",
			body:             "<frameset>[[ページ1]]</style>[[ページ2]]",
			currentTopicName: "トピックA",
			want:             nil,
		},
		{
			name:             "入れ子の a 要素の開始タグは開いている a 要素を閉じる",
			body:             `<a href="https://example.com/">[[ページ1]]<a href="https://example.com/">[[ページ2]]</a>[[ページ3]]`,
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ3]]", topicName: "トピックA", pageTitle: "ページ3"},
			},
		},
		{
			name:             "raw HTML の構文は返さず通常要素の本文にあるリンクは返す",
			body:             `<div data-page="[[ページ1]]">[[ページ2]]</div>` + "\n\n<!-- [[ページ3]] -->\n\n[[ページ4]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
				{text: "[[ページ4]]", topicName: "トピックA", pageTitle: "ページ4"},
			},
		},
		{
			name:             "Markdown リンクのリンク先にあるリンクは返さない",
			body:             "[外部](https://example.com/[[ページ1]]) と [[ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			name:             "山括弧付き Markdown リンク先の丸括弧より後ろも既存リンクとして保護する",
			body:             "[外部](<https://example.com/a)/[[ページ1]]>) と [[ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			name:             "山括弧付き Markdown リンク先の開き丸括弧も既存リンクとして保護する",
			body:             "[外部](<https://example.com/a([[ページ1]]>) と [[ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			name:             "ラベルのコードスパンにある角括弧はラベルを閉じない",
			body:             "[コード `]` と [[ページ1]]](https://example.com/) と [[ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			name:             "画像のラベルのコードスパンにある角括弧もラベルを閉じない",
			body:             "![`]`[[ページ1]]](https://example.com/) と [[ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			name:             "参照リンクのラベルのコードスパンにある角括弧もラベルを閉じない",
			body:             "[`]` と [[ページ1]]][参照] と [[ページ2]]\n\n[参照]: https://example.com/",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			name:             "Markdown リンクのラベルにあるリンクは返さない",
			body:             "[参照 [[ページ1]]](https://example.com/) と [[ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			name:             "リンク参照定義のリンク先にあるリンクは返さない",
			body:             "[a][ref]\n\n[ref]: /url/[[ページ1]]\n\n[[ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			name:             "リンク参照定義のタイトルにあるリンクは返さない",
			body:             "[a][ref]\n\n[ref]: /url \"[[ページ1]]\"\n\n[[ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			// ショートカット参照の後ろの [...] はリンクの一部ではないため、
			// 表示側と同じく通常の Wiki リンクとして扱う
			name:             "ショートカット参照の直後のリンクは返す",
			body:             "[a][[ページ1]]\n\n[a]: /url",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ1]]", topicName: "トピックA", pageTitle: "ページ1"},
			},
		},
		{
			name:             "raw HTML の a 要素の中のリンクは返さない",
			body:             `<a href="https://example.com/">[[ページ1]]</a> と [[ページ2]]`,
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			name:             "サニタイズで除去される a 要素の異なる終了タグ後は通常テキストとして返す",
			body:             "<a>[[ページ1]]</code>[[ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ1]]", topicName: "トピックA", pageTitle: "ページ1"},
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			name:             "raw HTML の script 要素の中のリンクは返さない",
			body:             `<script>const page = "[[ページ1]]"</script>` + "\n\n[[ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			name:             "raw HTML の style 要素の中のリンクは返さない",
			body:             `<style>/* [[ページ1]] */</style>` + "\n\n[[ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			// コードブロックに見せている raw text 系のタグでトークナイザーが本文の残りを
			// 飲み込むと、後ろの code 要素が認識されずリンクが返ってしまう
			name:             "コードブロックに raw text の要素を見せていても後続の code 要素の中のリンクは返さない",
			body:             "```html\n<style>\n```\n\n<code>[[ページ1]]</code> と [[ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			name:             "中身の無いリンクは返さない",
			body:             "[[]] と [[ページ1]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ1]]", topicName: "トピックA", pageTitle: "ページ1"},
			},
		},
		{
			// img タグだけの行は HTML ブロックを開かないため、続く行も画面と同じく Markdown として
			// 読まれる。コードスパンと既存リンクのラベルはそこでも保護される
			name: "img タグの行に続くコードと既存リンクのラベルは保護する",
			body: `<img src="https://example.com/i.png">` + "\n*キャプション*\n" +
				"`[[ページ1]]` と [参照 [[ページ2]]](https://example.com/) と [[ページ3]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ3]]", topicName: "トピックA", pageTitle: "ページ3"},
			},
		},
		{
			// サニタイズが受け付けないリンク先の a 要素は落ち、ラベルは通常テキストとして画面に出る
			name:             "サニタイズが落とすリンクのラベルにあるリンクは返す",
			body:             "[ラベル [[ページ1]]](tel:+81-3-0000-0000) と [[ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ1]]", topicName: "トピックA", pageTitle: "ページ1"},
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			name:             "リンク先が空のリンクのラベルにあるリンクは返す",
			body:             "[ラベル [[ページ1]]]()",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ1]]", topicName: "トピックA", pageTitle: "ページ1"},
			},
		},
		{
			name:             "サニタイズが落とす参照リンクのラベルにあるリンクは返す",
			body:             "[ラベル [[ページ1]]][参照]\n\n[参照]: javascript:alert",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ1]]", topicName: "トピックA", pageTitle: "ページ1"},
			},
		},
		{
			// ラベルの外はリンクが落ちても画面に出ないため、引き続き保護する
			name:             "サニタイズが落とすリンクのリンク先にあるリンクは返さない",
			body:             "[ラベル](tel:[[ページ1]]) と [[ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			// 画像のラベルは alt 属性になるため、リンク先によらずテキストにならない
			name:             "サニタイズが src を落とす画像のラベルにあるリンクは返さない",
			body:             "![ラベル [[ページ1]]](javascript:alert) と [[ページ2]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ2]]", topicName: "トピックA", pageTitle: "ページ2"},
			},
		},
		{
			// エスケープと文字参照は角括弧そのものであって、リンクを開く構文ではない。
			// ScanWikilinks の読み方と揃える
			name: "エスケープや文字参照で書いた角括弧のリンクは返さない",
			body: `\[\[ページ1]]` + "\n\n&#91;&#91;ページ2]]\n\n[[ページ3&#93;&#93;\n\n" +
				"&#x5B;&#x5B;ページ4]]\n\n[[ページ5]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ5]]", topicName: "トピックA", pageTitle: "ページ5"},
			},
		},
		{
			name:             "ページタイトルの文字参照は原文のまま扱う",
			body:             "[[ページ&amp;1]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ&amp;1]]", topicName: "トピックA", pageTitle: "ページ&amp;1"},
			},
		},
		{
			name:             "Wiki リンクを含まない本文では何も返さない",
			body:             "リンクの無い本文。",
			currentTopicName: "トピックA",
			want:             nil,
		},
		{
			name:             "対応の取れていない開始括弧とコードスパンの後ろのリンクを返す",
			body:             "[[ と書く `x` [[ページ1]] を見る",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ1]]", topicName: "トピックA", pageTitle: "ページ1"},
			},
		},
		{
			name:             "対応の取れていない開始括弧と Markdown リンクの後ろのリンクを返す",
			body:             "[[ と書く [リンク](/u) [[ページ1]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ1]]", topicName: "トピックA", pageTitle: "ページ1"},
			},
		},
		{
			name:             "対応の取れていない開始括弧と自動リンクの後ろのリンクを返す",
			body:             "[[ と書く <https://example.com/> [[ページ1]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ1]]", topicName: "トピックA", pageTitle: "ページ1"},
			},
		},
		{
			name:             "対応の取れていない開始括弧と raw HTML タグの後ろのリンクを返す",
			body:             "[[ と書く <b>太字</b> [[ページ1]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ1]]", topicName: "トピックA", pageTitle: "ページ1"},
			},
		},
		{
			name:             "保護範囲をまたぐリンクは返さない",
			body:             "[[ページ`x`1]]",
			currentTopicName: "トピックA",
			want:             nil,
		},
		{
			name:             "閉じていない a 要素を自動リンクが閉じた後のリンクを返す",
			body:             `<a href="/u">x <https://example.com> [[ページ1]]`,
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ1]]", topicName: "トピックA", pageTitle: "ページ1"},
			},
		},
		{
			name:             "閉じていない a 要素を Markdown リンクが閉じた後のリンクを返す",
			body:             `<a href="/u">x [リンク](/v) [[ページ1]]`,
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ1]]", topicName: "トピックA", pageTitle: "ページ1"},
			},
		},
		{
			name:             "閉じていない a 要素をメールの自動リンクが閉じた後のリンクを返す",
			body:             `<a href="/u">x <user@example.com> [[ページ1]]`,
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ1]]", topicName: "トピックA", pageTitle: "ページ1"},
			},
		},
		{
			name:             "閉じていない a 要素の中の画像の後ろのリンクは返さない",
			body:             `<a href="/u">x ![図](/v) [[ページ1]]`,
			currentTopicName: "トピックA",
			want:             nil,
		},
		{
			name:             "閉じていない a 要素をサニタイズが落とすリンクは閉じない",
			body:             `<a href="/u">x [リンク](javascript:x) [[ページ1]]`,
			currentTopicName: "トピックA",
			want:             nil,
		},
		{
			name:             "閉じていない code 要素は自動リンクでは閉じない",
			body:             "x <code>y <https://example.com> [[ページ1]]",
			currentTopicName: "トピックA",
			want:             nil,
		},
		{
			name:             "引用の中の閉じていない pre 要素は引用の外を保護しない",
			body:             "> x <pre>y\n\n> z [[ページ1]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ1]]", topicName: "トピックA", pageTitle: "ページ1"},
			},
		},
		{
			name:             "見出しの中の閉じていない pre 要素は見出しの外を保護しない",
			body:             "# x <pre>y\n\nz [[ページ1]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ1]]", topicName: "トピックA", pageTitle: "ページ1"},
			},
		},
		{
			name:             "表のセルの中の閉じていない code 要素はセルの外を保護しない",
			body:             "| a |\n| --- |\n| <code>y |\n\nz [[ページ1]]",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ1]]", topicName: "トピックA", pageTitle: "ページ1"},
			},
		},
		{
			name:             "表のセルの中の閉じていない a 要素は次のセルを保護しない",
			body:             "| a | b |\n| --- | --- |\n| <a href=\"/u\">y | z [[ページ1]] |",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ1]]", topicName: "トピックA", pageTitle: "ページ1"},
			},
		},
		{
			name:             "段落の中の閉じていない pre 要素は後続の段落も保護する",
			body:             "x <pre>y\n\nz [[ページ1]]",
			currentTopicName: "トピックA",
			want:             nil,
		},
		{
			name:             "引用の中の閉じていない code 要素は引用の外も保護する",
			body:             "> x <code>y\n\n> z [[ページ1]]",
			currentTopicName: "トピックA",
			want:             nil,
		},
		{
			name:             "表のセルの中の閉じていない script 要素は表の外も保護する",
			body:             "| a |\n| --- |\n| <script>y |\n\nz [[ページ1]]",
			currentTopicName: "トピックA",
			want:             nil,
		},
		{
			name:             "HTML として読まれなかったタグの中のリンクを返す",
			body:             "before <!-- [[ページ1]] </script> after",
			currentTopicName: "トピックA",
			want: []want{
				{text: "[[ページ1]]", topicName: "トピックA", pageTitle: "ページ1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			matches := ScanWikilinkMatches(tt.body, tt.currentTopicName)

			if len(matches) != len(tt.want) {
				t.Fatalf("len(matches) = %d, want %d", len(matches), len(tt.want))
			}

			for i, w := range tt.want {
				got := matches[i]

				if text := tt.body[got.Start:got.Stop]; text != w.text {
					t.Errorf("matches[%d] の範囲の文字列 = %q, want %q", i, text, w.text)
				}
				if got.Key.TopicName != w.topicName {
					t.Errorf("matches[%d].Key.TopicName = %q, want %q", i, got.Key.TopicName, w.topicName)
				}
				if got.Key.PageTitle != w.pageTitle {
					t.Errorf("matches[%d].Key.PageTitle = %q, want %q", i, got.Key.PageTitle, w.pageTitle)
				}
			}
		})
	}
}

func TestScanWikilinkMatches_MultilineDestinations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		link string
	}{
		{name: "引用内の閉じ丸括弧", link: "> [x](\n> <https://example.com/a)/[[A]]>)"},
		{name: "引用内の開き丸括弧", link: "> [x](\n> <https://example.com/a([[A]]>)"},
		{name: "CRLF の引用", link: "> [x](\r\n> <https://example.com/a)/[[A]]>)"},
		{name: "入れ子の引用", link: "> > [x](\n> > <https://example.com/a([[A]]>)"},
		{name: "引用内のリスト", link: "> - [x](\n>   <https://example.com/a)/[[A]]>)"},
		{name: "リストの継続行", link: "- [x](\n  <https://example.com/a([[A]]>)"},
		{name: "タイトル付きの引用", link: "> [x](\n> <https://example.com/a)/[[A]]> \"title ) [[A]]\")"},
		{name: "引用内の画像", link: "> ![x](\n> <https://example.com/a([[A]]>)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			body := tt.link + " [[B]]"
			matches := ScanWikilinkMatches(body, "T")
			want := []string{"B"}

			if len(matches) != len(want) {
				t.Fatalf("matches = %+v, want %v", matches, want)
			}
			for i, match := range matches {
				if body[match.Start:match.Stop] != "[["+want[i]+"]]" ||
					match.Key.TopicName != "T" || match.Key.PageTitle != want[i] {
					t.Errorf("match[%d] = %+v, want [[%s]] in topic T", i, match, want[i])
				}
			}

		})
	}
}

func TestScanWikilinkMatches_DestinationEscapes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		destination string
		keepsLink   bool
	}{
		{name: "通常の許可 URL", destination: "https://example.com/[[A]]", keepsLink: true},
		{name: "エスケープした許可 URL", destination: `https\://example.com/[[A]]`, keepsLink: true},
		{name: "数値文字参照の許可 URL", destination: "https&#58;//example.com/[[A]]", keepsLink: true},
		{name: "名前付き文字参照の許可 URL", destination: "https&colon;//example.com/[[A]]", keepsLink: true},
		{name: "通常の拒否 URL", destination: "tel:+81-3-0000-0000/[[A]]"},
		{name: "エスケープした拒否 URL", destination: `tel\:+81-3-0000-0000/[[A]]`},
		{name: "数値文字参照の拒否 URL", destination: "tel&#58;+81-3-0000-0000/[[A]]"},
		{name: "名前付き文字参照の拒否 URL", destination: "tel&colon;+81-3-0000-0000/[[A]]"},
	}

	for _, tt := range tests {
		for _, reference := range []bool{false, true} {
			name := tt.name
			if reference {
				name += "/参照リンク"
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				body := "[x [[A]]](" + tt.destination + ") [[B]]"
				if reference {
					body = "[x [[A]]][ref] [[B]]\n\n[ref]: " + tt.destination
				}
				if keepsLink := strings.Contains(RenderMarkdown(body), "<a "); keepsLink != tt.keepsLink {
					t.Fatalf("rendered anchor exists = %v, want %v", keepsLink, tt.keepsLink)
				}
				matches := ScanWikilinkMatches(body, "T")
				want := []string{"B"}
				if !tt.keepsLink {
					want = []string{"A", "B"}
				}

				if len(matches) != len(want) {
					t.Fatalf("matches = %+v, want %v", matches, want)
				}
				for i, match := range matches {
					if body[match.Start:match.Stop] != "[["+want[i]+"]]" ||
						match.Key.TopicName != "T" || match.Key.PageTitle != want[i] {
						t.Errorf("match[%d] = %+v, want [[%s]] in topic T", i, match, want[i])
					}
				}

			})
		}
	}
}

func TestScanWikilinkMatches_EscapedOpening(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want []string
	}{
		{name: "開始括弧のエスケープ", body: "\\[[A]] と [[B]]", want: []string{"B"}},
		{name: "バックスラッシュのエスケープ", body: "\\\\[[A]] と [[B]]", want: []string{"A", "B"}},
		{name: "3 本のバックスラッシュ", body: "\\\\\\[[A]] と [[B]]", want: []string{"B"}},
		{name: "両方の開始括弧のエスケープ", body: "\\[\\[A]] と [[B]]", want: []string{"B"}},
		{name: "文字参照の開始括弧", body: "&#91;&#91;A]] &lbrack;&lbrack;A]] [[B]]", want: []string{"B"}},
		{name: "HTML ブロック内のバックスラッシュ", body: "<div>\n\\[[A]]\n</div>\n\n[[B]]", want: []string{"A", "B"}},
		{name: "HTML ブロック内の 3 本のバックスラッシュ", body: "<div>\n\\\\\\[[A]]\n</div>\n\n[[B]]", want: []string{"A", "B"}},
		{name: "引用の HTML ブロック", body: "> <div>\n> \\[[A]]\n> </div>\n\n[[B]]", want: []string{"A", "B"}},
		{name: "HTML ブロックの後の Markdown", body: "<div>\\[[A]]</div>\n\n\\[[A]] [[B]]", want: []string{"A", "B"}},
		{name: "インライン HTML 内の Markdown", body: "本文 <span>\\[[A]]</span> [[B]]", want: []string{"B"}},
		{name: "拒否されたリンクのラベル", body: "[x \\[[A]]](tel:+81-3-0000-0000) [[B]]", want: []string{"B"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			body, want := tt.body, tt.want
			matches := ScanWikilinkMatches(body, "T")

			if len(matches) != len(want) {
				t.Fatalf("matches = %+v, want %v", matches, want)
			}
			for i, match := range matches {
				if body[match.Start:match.Stop] != "[["+want[i]+"]]" ||
					match.Key.TopicName != "T" || match.Key.PageTitle != want[i] {
					t.Errorf("match[%d] = %+v, want [[%s]] in topic T", i, match, want[i])
				}
			}

		})
	}
}

func TestScanWikilinkMatches_ElementScopeEndTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want []string
	}{
		{name: "囲む要素の終了タグが中の pre を終わらせる", body: "<div><pre></div>[[A]]", want: []string{"A"}},
		{name: "段落をまたぐ囲む要素の終了タグ", body: "<div><pre></div>\n\n[[A]]", want: []string{"A"}},
		{name: "対応しない終了タグは何も閉じない", body: "<h1></li><pre></h1>[[A]]", want: []string{"A"}},
		{name: "同じ形が続いても独立して閉じる", body: "<div><pre></div>[[A]] <div><pre></div>[[B]]", want: []string{"A", "B"}},
		{name: "閉じた後の余分な終了タグ", body: "<div><pre></div>[[A]]</pre>[[B]]", want: []string{"A", "B"}},
		{name: "リスト項目の終了タグ", body: "<ul><li><pre></li>[[A]]", want: []string{"A"}},
		{name: "formatting element は開き直される", body: "<div><code></div>[[A]]", want: nil},
		{name: "段落の終了タグと formatting element", body: "<p><code></p>[[A]]", want: nil},
		{name: "表の後の終了タグは手前へ届かない", body: "<div></div><code><table></code>[[A]]", want: nil},
		{name: "表の後の pre の終了タグ", body: "<em><pre><table></pre>[[A]]", want: nil},
		{name: "規則を持たない終了タグは special 要素で止まる", body: "<span><pre></span>[[A]]", want: nil},
		{name: "表の外のセルは何も開かない", body: `<code><td><img src="/x"></code>[[A]]`, want: []string{"A"}},
		{name: "表の中のセルの終了タグ", body: "<table><td><code>[[A]]</code></td></table>[[B]]", want: []string{"B"}},
		{name: "raw text の中の Markdown リンクは a を閉じない", body: `<a href="/x"><textarea>[l](/x)[[A]]`, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			body, want := tt.body, tt.want
			matches := ScanWikilinkMatches(body, "T")

			if len(matches) != len(want) {
				t.Fatalf("matches = %+v, want %v", matches, want)
			}
			for i, match := range matches {
				if body[match.Start:match.Stop] != "[["+want[i]+"]]" ||
					match.Key.TopicName != "T" || match.Key.PageTitle != want[i] {
					t.Errorf("match[%d] = %+v, want [[%s]] in topic T", i, match, want[i])
				}
			}
		})
	}
}

func TestScanWikilinks_SkipsCode(t *testing.T) {
	t.Parallel()

	// A [[...]] written as code is not a link on the screen, so the save paths must not create
	// the page it names. Links outside code keep resolving as before.
	//
	// [Ja] コードとして書かれた [[...]] は画面上でリンクにならないため、保存経路がその名前の
	// ページを作ってはならない。コードの外のリンクは従来どおり解決される。
	tests := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "フェンスコードブロック内のリンクは返さない",
			body: "[[ページ1]]\n\n```\n[[コード1]]\n```\n\n[[ページ2]]",
			want: []string{"ページ1", "ページ2"},
		},
		{
			name: "インデントコードブロック内のリンクは返さない",
			body: "[[ページ1]]\n\n    [[コード1]]\n\n[[ページ2]]",
			want: []string{"ページ1", "ページ2"},
		},
		{
			name: "インラインコード内のリンクは返さない",
			body: "[[ページ1]] `[[コード1]]` [[トピックB/ページ2]]",
			want: []string{"ページ1", "トピックB/ページ2"},
		},
		{
			name: "raw HTML の code 要素内のリンクは返さない",
			body: "[[ページ1]] <code>[[コード1]]</code> [[ページ2]]",
			want: []string{"ページ1", "ページ2"},
		},
		{
			name: "raw HTML の pre 要素内のリンクは返さない",
			body: "[[ページ1]]\n\n<pre>\n[[コード1]]\n</pre>\n\n[[ページ2]]",
			want: []string{"ページ1", "ページ2"},
		},
		{
			name: "コードだけの本文からは何も返さない",
			body: "```\n[[コード1]]\n```",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			keys := ScanWikilinks(tt.body, "トピックA")

			if len(keys) != len(tt.want) {
				t.Fatalf("len(keys) = %d, want %d: %+v", len(keys), len(tt.want), keys)
			}
			for i, raw := range tt.want {
				if keys[i].Raw != raw {
					t.Errorf("keys[%d].Raw = %q, want %q", i, keys[i].Raw, raw)
				}
			}
		})
	}
}

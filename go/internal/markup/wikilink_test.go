package markup

import (
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

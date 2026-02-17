package markup

import (
	"regexp"
	"strings"
	"testing"
)

// normalizeHTML はHTML文字列の空白を正規化する（テスト比較用）
func normalizeHTML(html string) string {
	re := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(re.ReplaceAllString(html, " "))
}

func TestRenderMarkdown_EmptyString(t *testing.T) {
	t.Parallel()

	got := RenderMarkdown("")
	if got != "" {
		t.Errorf("RenderMarkdown(\"\") = %q, want \"\"", got)
	}
}

func TestRenderMarkdown_BasicMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "見出し",
			input:    "# Title",
			expected: "<h1 id=\"title\">Title</h1>",
		},
		{
			name:     "段落",
			input:    "これはテストです。",
			expected: "<p>これはテストです。</p>",
		},
		{
			name:     "太字",
			input:    "**太字テキスト**",
			expected: "<p><strong>太字テキスト</strong></p>",
		},
		{
			name:     "イタリック",
			input:    "*イタリック*",
			expected: "<p><em>イタリック</em></p>",
		},
		{
			name:     "リンク",
			input:    "[リンク](https://example.com)",
			expected: `<p><a href="https://example.com" rel="nofollow">リンク</a></p>`,
		},
		{
			name:     "画像",
			input:    "![alt text](https://example.com/image.png)",
			expected: `<p><img src="https://example.com/image.png" alt="alt text"></p>`,
		},
		{
			name:     "インラインコード",
			input:    "`code`",
			expected: "<p><code>code</code></p>",
		},
		{
			name:     "コードブロック",
			input:    "```\ncode block\n```",
			expected: "<pre><code>code block\n</code></pre>",
		},
		{
			name:     "順序なしリスト",
			input:    "- item1\n- item2",
			expected: "<ul> <li>item1</li> <li>item2</li> </ul>",
		},
		{
			name:     "順序付きリスト",
			input:    "1. first\n2. second",
			expected: "<ol> <li>first</li> <li>second</li> </ol>",
		},
		{
			name:     "引用",
			input:    "> 引用テキスト",
			expected: "<blockquote> <p>引用テキスト</p> </blockquote>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := normalizeHTML(RenderMarkdown(tt.input))
			want := normalizeHTML(tt.expected)
			if got != want {
				t.Errorf("RenderMarkdown(%q)\ngot:  %s\nwant: %s", tt.input, got, want)
			}
		})
	}
}

func TestRenderMarkdown_GFMExtensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "タスクリスト（未完了）",
			input:    "- [ ] 未完了\n- [x] 完了",
			expected: `<ul> <li><input disabled="" type="checkbox"> 未完了</li> <li><input checked="" disabled="" type="checkbox"> 完了</li> </ul>`,
		},
		{
			name:     "打ち消し線",
			input:    "~~削除~~",
			expected: "<p><del>削除</del></p>",
		},
		{
			name:     "テーブル",
			input:    "| A | B |\n| --- | --- |\n| 1 | 2 |",
			expected: `<table> <thead> <tr> <th>A</th> <th>B</th> </tr> </thead> <tbody> <tr> <td>1</td> <td>2</td> </tr> </tbody> </table>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := normalizeHTML(RenderMarkdown(tt.input))
			want := normalizeHTML(tt.expected)
			if got != want {
				t.Errorf("RenderMarkdown(%q)\ngot:  %s\nwant: %s", tt.input, got, want)
			}
		})
	}
}

func TestRenderMarkdown_HardWraps(t *testing.T) {
	t.Parallel()

	input := "行1\n行2"
	got := RenderMarkdown(input)

	if !strings.Contains(got, "<br") {
		t.Errorf("RenderMarkdown(%q) should contain <br>, got: %s", input, got)
	}
}

func TestRenderMarkdown_UnsafeHTML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "HTMLタグがそのまま出力される",
			input:    "<div>HTMLコンテンツ</div>",
			contains: "HTMLコンテンツ",
		},
		{
			name:     "imgタグが保持される",
			input:    `<img src="https://example.com/image.png" alt="test" width="100" height="50">`,
			contains: `<img src="https://example.com/image.png" alt="test" width="100" height="50"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := RenderMarkdown(tt.input)
			if !strings.Contains(got, tt.contains) {
				t.Errorf("RenderMarkdown(%q)\ngot:  %s\nwant to contain: %s", tt.input, got, tt.contains)
			}
		})
	}
}

func TestRenderMarkdown_Sanitization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		notContains string
	}{
		{
			name:        "scriptタグが除去される",
			input:       "<script>alert('xss')</script>",
			notContains: "<script>",
		},
		{
			name:        "onclickイベントハンドラが除去される",
			input:       `<div onclick="alert('xss')">click</div>`,
			notContains: "onclick",
		},
		{
			name:        "javascriptプロトコルが除去される",
			input:       `<a href="javascript:alert('xss')">link</a>`,
			notContains: "javascript:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := RenderMarkdown(tt.input)
			if strings.Contains(got, tt.notContains) {
				t.Errorf("RenderMarkdown(%q) should not contain %q, got: %s", tt.input, tt.notContains, got)
			}
		})
	}
}

func TestRenderMarkdown_ImgAttributes(t *testing.T) {
	t.Parallel()

	input := `<img src="https://example.com/image.png" alt="test" width="200" height="100">`
	got := RenderMarkdown(input)

	checks := []string{
		`width="200"`,
		`height="100"`,
	}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Errorf("RenderMarkdown(%q) should contain %q, got: %s", input, check, got)
		}
	}
}

func TestRenderMarkdown_StandaloneImgZWNJ(t *testing.T) {
	t.Parallel()

	// 単独行のimgタグの後にイタリック記法がある場合、正しく処理されること
	input := "<img src=\"https://example.com/image.png\">\n*caption*"
	got := RenderMarkdown(input)

	// ZWNJが残っていないことを確認
	if strings.Contains(got, zwnj) {
		t.Errorf("RenderMarkdown(%q) should not contain ZWNJ, got: %s", input, got)
	}

	// imgタグが出力されていること
	if !strings.Contains(got, "<img") {
		t.Errorf("RenderMarkdown(%q) should contain <img>, got: %s", input, got)
	}

	// イタリック記法が変換されていること
	if !strings.Contains(got, "<em>") {
		t.Errorf("RenderMarkdown(%q) should contain <em>, got: %s", input, got)
	}
}

func TestRenderMarkdown_TaskListCheckboxAttributes(t *testing.T) {
	t.Parallel()

	input := "- [ ] タスク"
	got := RenderMarkdown(input)

	// input要素が保持されていること（bluemondayで除去されないこと）
	if !strings.Contains(got, "<input") {
		t.Errorf("RenderMarkdown(%q) should contain <input>, got: %s", input, got)
	}

	// disabled属性が保持されていること
	if !strings.Contains(got, "disabled") {
		t.Errorf("RenderMarkdown(%q) should contain disabled attribute, got: %s", input, got)
	}

	// type="checkbox"が保持されていること
	if !strings.Contains(got, `type="checkbox"`) {
		t.Errorf("RenderMarkdown(%q) should contain type=\"checkbox\", got: %s", input, got)
	}
}

func TestRenderMarkdown_ComplexDocument(t *testing.T) {
	t.Parallel()

	input := `# ページタイトル

これは**太字**と*イタリック*を含む段落です。

## セクション1

- リスト項目1
- リスト項目2

### タスクリスト

- [ ] 未完了のタスク
- [x] 完了したタスク

> 引用文

` + "```go\nfunc main() {\n}\n```"

	got := RenderMarkdown(input)

	checks := []string{
		"<h1",
		"<strong>太字</strong>",
		"<em>イタリック</em>",
		"<h2",
		"<li>リスト項目1</li>",
		`<input disabled="" type="checkbox">`,
		`<input checked="" disabled="" type="checkbox">`,
		"<blockquote>",
		"<pre><code",
	}

	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Errorf("RenderMarkdown should contain %q, got: %s", check, got)
		}
	}
}

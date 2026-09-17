package markup

import (
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// TestRenderHTML_ShiftsHeadingsは、本文の最初の見出しがh2になるよう、すべての見出しが
// 同じ段数だけずれて描画されることを固定する。ページ表示画面はページタイトルをh1として描画する
// ため、本文からはh1が出てはならず、本文の見出しはh2から飛ばずに始まらなければならない。
func TestRenderHTML_ShiftsHeadings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "#から書いた本文は1段下がる",
			body: "# 1\n\n## 2\n\n### 3\n\n#### 4",
			want: []string{`<h2 id="1">1</h2>`, `<h3 id="2">2</h3>`, `<h4 id="3">3</h4>`, `<h5 id="4">4</h5>`},
		},
		{
			name: "##から書いた本文はそのまま",
			body: "## 2\n\n### 3\n\n#### 4",
			want: []string{`<h2 id="2">2</h2>`, `<h3 id="3">3</h3>`, `<h4 id="4">4</h4>`},
		},
		{
			name: "###から書いた本文は1段上がる",
			body: "### 3\n\n#### 4\n\n###### 6",
			want: []string{`<h2 id="3">3</h2>`, `<h3 id="4">4</h3>`, `<h5 id="6">6</h5>`},
		},
		{
			name: "h6より深くなる見出しはh6に潰れる",
			body: "# 1\n\n##### 5\n\n###### 6",
			want: []string{`<h2 id="1">1</h2>`, `<h6 id="5">5</h6>`, `<h6 id="6">6</h6>`},
		},
		{
			name: "途中でレベルを飛ばした段差はそのまま残る",
			body: "## 2\n\n#### 4",
			want: []string{`<h2 id="2">2</h2>`, `<h4 id="4">4</h4>`},
		},
		{
			name: "途中に最初の見出しより浅い#があっても、他の見出しはずれずその#がh2に潰れる",
			body: "## 節\n\n# 例\n\n### 小節",
			want: []string{`<h2 id="節">節</h2>`, `<h2 id="例">例</h2>`, `<h3 id="小節">小節</h3>`},
		},
		{
			name: "###から書いた本文の途中の#と##はh2に潰れる",
			body: "### 3\n\n# 1\n\n## 2\n\n#### 4",
			want: []string{`<h2 id="3">3</h2>`, `<h2 id="1">1</h2>`, `<h2 id="2">2</h2>`, `<h3 id="4">4</h3>`},
		},
		{
			name: "本文に直接書かれたh1も最初の見出しの判定に含まれる",
			body: "<h1>生のh1</h1>\n\n## 2",
			want: []string{"<h2>生のh1</h2>", `<h3 id="2">2</h3>`},
		},
		{
			name: "引用の中の見出しも最初の見出しの判定とずらす対象に含まれる",
			body: "> # 引用の中\n\n## 2",
			want: []string{"<blockquote>", `<h2 id="引用の中">引用の中</h2>`, `<h3 id="2">2</h3>`},
		},
		{
			name: "見出しのidは保たれ、ページ内アンカーが指す先が変わらない",
			body: "### Section\n\n[飛ぶ](#section)",
			want: []string{`<h2 id="section">Section</h2>`, `href="#section"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := RenderHTML(
				context.Background(),
				tt.body,
				"topic1",
				"space-1",
				"my-space",
				&mockPageLocationResolver{},
				&mockBatchAttachmentFinder{},
			)
			if err != nil {
				t.Fatalf("予期しないエラー: %v", err)
			}

			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("結果に%qが含まれていない: %s", want, got)
				}
			}
			if strings.Contains(got, "<h1") {
				t.Errorf("本文の結果にh1が含まれている: %s", got)
			}
		})
	}
}

// TestRenderHTML_ShiftsHeadingsWithWikilinksは、Wikiリンクを含む本文でも見出しがずれる
// ことを固定する。リンクの解決は本文をもう一度レンダリングしてツリーを差し替えるため、ずらす
// 工程がその差し替えより後ろにあることを確かめる意味がある。
func TestRenderHTML_ShiftsHeadingsWithWikilinks(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{
		locations: []PageLocation{
			{
				Key:        WikilinkKey{Raw: "ページA", TopicName: "topic1", PageTitle: "ページA"},
				TopicName:  "topic1",
				PageID:     model.PageID("page-1"),
				PageNumber: 1,
				PageTitle:  "ページA",
			},
		},
	}

	got, err := RenderHTML(
		context.Background(),
		"# [[ページA]]",
		"topic1",
		"space-1",
		"my-space",
		resolver,
		&mockBatchAttachmentFinder{},
	)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	if !strings.Contains(got, `<h2 id="ページa"><a href="/s/my-space/pages/1">ページA</a></h2>`) {
		t.Errorf("見出しの中のWikiリンクがh2として描画されていない: %s", got)
	}
	if strings.Contains(got, "<h1") {
		t.Errorf("本文の結果にh1が含まれている: %s", got)
	}
}

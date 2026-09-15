package exportfile

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestWithFrontmatter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		body  string
		title string
		want  string
	}{
		{
			name:  "本文の前にfrontmatterのブロックを置く",
			body:  "本文がここから始まる。\n",
			title: "設計メモ",
			want:  "---\nwikino_title: \"設計メモ\"\n---\n\n本文がここから始まる。\n",
		},
		{
			name:  "空の本文でもfrontmatterを持たせる",
			body:  "",
			title: "設計メモ",
			want:  "---\nwikino_title: \"設計メモ\"\n---\n\n",
		},
		{
			name:  "記号を含むタイトルをYAMLの文字列としてエスケープする",
			body:  "本文\n",
			title: `Annict | "見たアニメ" \ 記録`,
			want:  "---\nwikino_title: \"Annict | \\\"見たアニメ\\\" \\\\ 記録\"\n---\n\n本文\n",
		},
		{
			name:  "HTMLの記号はエスケープせずそのまま残す",
			body:  "本文\n",
			title: "a < b & c > d",
			want:  "---\nwikino_title: \"a < b & c > d\"\n---\n\n本文\n",
		},
		{
			name:  "既にあるfrontmatterのブロックへ差し込む",
			body:  "---\naliases:\n  - 別名\n---\n\n本文\n",
			title: "設計メモ",
			want:  "---\nwikino_title: \"設計メモ\"\naliases:\n  - 別名\n---\n\n本文\n",
		},
		{
			name:  "CRLFのfrontmatterのブロックへも差し込む",
			body:  "---\r\naliases: []\r\n---\r\n\r\n本文\r\n",
			title: "設計メモ",
			want:  "---\r\nwikino_title: \"設計メモ\"\r\naliases: []\r\n---\r\n\r\n本文\r\n",
		},
		{
			name:  "CRLFの本文へ置くブロックもCRLFで書く",
			body:  "本文がここから始まる。\r\n続き\r\n",
			title: "設計メモ",
			want:  "---\r\nwikino_title: \"設計メモ\"\r\n---\r\n\r\n本文がここから始まる。\r\n続き\r\n",
		},
		{
			name:  "閉じられていない区切りはfrontmatterではなく水平線として扱う",
			body:  "---\n\n本文\n",
			title: "設計メモ",
			want:  "---\nwikino_title: \"設計メモ\"\n---\n\n---\n\n本文\n",
		},
		{
			name:  "区切りで始まらない本文には新しいブロックを置く",
			body:  "# 見出し\n\n---\n",
			title: "設計メモ",
			want:  "---\nwikino_title: \"設計メモ\"\n---\n\n# 見出し\n\n---\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := WithFrontmatter(tt.body, tt.title); got != tt.want {
				t.Errorf("WithFrontmatter() = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}

func TestWithFrontmatter_ExistingMappings(t *testing.T) {
	t.Parallel()
	tests := []struct{ name, body, wantAlias string }{
		{"既存のキー", "---\nwikino_title: old\naliases: [other]\n---\n\nbody\n", "other"},
		{"引用符付きのキー", "---\n\"wikino_title\": old\naliases: [other]\n---\n\nbody\n", "other"},
		{"フロー形式のマッピング", "---\n{wikino_title: old, aliases: [other]}\n---\n\nbody\n", "other"},
		{"BOM付きのマッピング", "---\n\uFEFFaliases: [other]\n---\n\nbody\n", "other"},
		{"フロー形式への差し込み", "---\n{aliases: [other]}\n---\n\nbody\n", "other"},
		{"タグ付きのマッピング", "---\n!!map\naliases: [other]\n---\n\nbody\n", "other"},
		{"アンカー付きのマッピング", "---\n&properties\naliases: [other]\n---\n\nbody\n", "other"},
		{"複数行の値", "---\nwikino_title: |\n  old\n  title\naliases: [other]\n---\n\nbody\n", "other"},
		{"アンカー付きの値", "---\nwikino_title: &title old\naliases: [*title]\n---\n\nbody\n", "current: \"title\""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := WithFrontmatter(tt.body, "current: \"title\"")
			if !strings.HasSuffix(got, "---\n\nbody\n") {
				t.Errorf("本文が変わった: %q", got)
			}
			opening, closing, _, ok := frontmatterBlock(got)
			if !ok {
				t.Fatalf("不正なフロントマター: %q", got)
			}
			var properties map[string]any
			if err := yaml.Unmarshal([]byte(got[opening:closing]), &properties); err != nil {
				t.Fatal(err)
			}
			if properties[titleKey] != "current: \"title\"" {
				t.Errorf("title=%v", properties[titleKey])
			}
			aliases, ok := properties["aliases"].([]any)
			if !ok || len(aliases) != 1 {
				t.Fatalf("エイリアスが失われた: %v", properties)
			}
			if aliases[0] != tt.wantAlias {
				t.Errorf("alias = %v、期待値 = %s", aliases[0], tt.wantAlias)
			}
			if count := strings.Count(got, "wikino_title"); count != 1 {
				t.Errorf("タイトルのキーの件数 = %d: %q", count, got)
			}
		})
	}
}

func TestWithFrontmatter_PreservesNonMappings(t *testing.T) {
	t.Parallel()
	for _, body := range []string{
		"---\n\nordinary paragraph\n\n---\n\nbody",
		"---\n- list item\n---\nbody",
		"---\ninvalid: [\n---\nbody",
		"---\na: 1\na: 2\n---\nbody",
		"---\n{}\n...\nstray content\n---\nbody",
	} {
		t.Run(body, func(t *testing.T) {
			t.Parallel()
			got := WithFrontmatter(body, "current")
			want := "---\nwikino_title: \"current\"\n---\n\n" + body
			if got != want {
				t.Errorf("本文がメタデータとして消費された: %q", got)
			}
		})
	}
}

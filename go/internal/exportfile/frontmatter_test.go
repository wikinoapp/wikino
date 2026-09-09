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
			name:  "本文の前に frontmatter のブロックを置く",
			body:  "本文がここから始まる。\n",
			title: "設計メモ",
			want:  "---\nwikino_title: \"設計メモ\"\n---\n\n本文がここから始まる。\n",
		},
		{
			name:  "空の本文でも frontmatter を持たせる",
			body:  "",
			title: "設計メモ",
			want:  "---\nwikino_title: \"設計メモ\"\n---\n\n",
		},
		{
			name:  "記号を含むタイトルを YAML の文字列としてエスケープする",
			body:  "本文\n",
			title: `Annict | "見たアニメ" \ 記録`,
			want:  "---\nwikino_title: \"Annict | \\\"見たアニメ\\\" \\\\ 記録\"\n---\n\n本文\n",
		},
		{
			name:  "HTML の記号はエスケープせずそのまま残す",
			body:  "本文\n",
			title: "a < b & c > d",
			want:  "---\nwikino_title: \"a < b & c > d\"\n---\n\n本文\n",
		},
		{
			name:  "既にある frontmatter のブロックへ差し込む",
			body:  "---\naliases:\n  - 別名\n---\n\n本文\n",
			title: "設計メモ",
			want:  "---\nwikino_title: \"設計メモ\"\naliases:\n  - 別名\n---\n\n本文\n",
		},
		{
			name:  "CRLF の frontmatter のブロックへも差し込む",
			body:  "---\r\naliases: []\r\n---\r\n\r\n本文\r\n",
			title: "設計メモ",
			want:  "---\r\nwikino_title: \"設計メモ\"\r\naliases: []\r\n---\r\n\r\n本文\r\n",
		},
		{
			name:  "CRLF の本文へ置くブロックも CRLF で書く",
			body:  "本文がここから始まる。\r\n続き\r\n",
			title: "設計メモ",
			want:  "---\r\nwikino_title: \"設計メモ\"\r\n---\r\n\r\n本文がここから始まる。\r\n続き\r\n",
		},
		{
			name:  "閉じられていない区切りは frontmatter ではなく水平線として扱う",
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
				t.Errorf("WithFrontmatter() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWithFrontmatter_ExistingMappings(t *testing.T) {
	t.Parallel()
	tests := []struct{ name, body string }{
		{"existing key", "---\nwikino_title: old\naliases: [other]\n---\n\nbody\n"},
		{"quoted key", "---\n\"wikino_title\": old\naliases: [other]\n---\n\nbody\n"},
		{"flow mapping", "---\n{wikino_title: old, aliases: [other]}\n---\n\nbody\n"},
		{"mapping with a BOM", "---\n\uFEFFaliases: [other]\n---\n\nbody\n"},
		{"flow insertion", "---\n{aliases: [other]}\n---\n\nbody\n"},
		{"tagged mapping", "---\n!!map\naliases: [other]\n---\n\nbody\n"},
		{"anchored mapping", "---\n&properties\naliases: [other]\n---\n\nbody\n"},
		{"multiline value", "---\nwikino_title: |\n  old\n  title\naliases: [other]\n---\n\nbody\n"},
		{"anchored value", "---\nwikino_title: &title old\naliases: [*title]\n---\n\nbody\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := WithFrontmatter(tt.body, "current: \"title\"")
			if !strings.HasSuffix(got, "---\n\nbody\n") {
				t.Errorf("body changed: %q", got)
			}
			opening, closing, _, ok := frontmatterBlock(got)
			if !ok {
				t.Fatalf("invalid frontmatter: %q", got)
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
				t.Fatalf("aliases lost: %v", properties)
			}
			wantAlias := "other"
			if tt.name == "anchored value" {
				wantAlias = "current: \"title\""
			}
			if aliases[0] != wantAlias {
				t.Errorf("alias=%v, want %s", aliases[0], wantAlias)
			}
			if count := strings.Count(got, "wikino_title"); count != 1 {
				t.Errorf("title key count=%d: %q", count, got)
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
				t.Errorf("body was consumed as metadata: %q", got)
			}
		})
	}
}

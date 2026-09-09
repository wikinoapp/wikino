package exportfile

import (
	"strconv"
	"strings"
	"testing"
)

func TestDeduperUnique(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		titles []string
		ext    string
		want   []string
	}{
		{
			name:   "衝突しない名前はそのまま返す",
			titles: []string{"api", "cli"},
			ext:    ".md",
			want:   []string{"api.md", "cli.md"},
		},
		{
			name:   "同じ名前が続くと連番を付ける",
			titles: []string{"api", "api", "api"},
			ext:    ".md",
			want:   []string{"api.md", "api-2.md", "api-3.md"},
		},
		{
			name:   "連番は拡張子の前に付ける",
			titles: []string{"設計メモ", "設計メモ"},
			ext:    ".md",
			want:   []string{"設計メモ.md", "設計メモ-2.md"},
		},
		{
			name:   "大文字小文字だけが違う名前も衝突とみなす",
			titles: []string{"API", "api"},
			ext:    ".md",
			want:   []string{"API.md", "api-2.md"},
		},
		{
			name:   "合成の仕方だけが違う名前も衝突とみなす",
			titles: []string{"が", "か\u3099"},
			ext:    ".md",
			want:   []string{"が.md", "か\u3099-2.md"},
		},
		{
			name:   "全角へ置き換えた結果が同じなら衝突とみなす",
			titles: []string{"API|一覧", "API｜一覧"},
			ext:    ".md",
			want:   []string{"API｜一覧.md", "API｜一覧-2.md"},
		},
		{
			name:   "連番と同じ名前がすでにあるときは次の数へ進む",
			titles: []string{"api", "api-2", "api"},
			ext:    ".md",
			want:   []string{"api.md", "api-2.md", "api-3.md"},
		},
		{
			name:   "拡張子が空でも連番を付ける",
			titles: []string{"Topic", "topic"},
			ext:    "",
			want:   []string{"Topic", "topic-2"},
		},
		{
			name:   "連番の分だけ切り詰めて上限バイト数に収める",
			titles: []string{strings.Repeat("あ", 100), strings.Repeat("あ", 100)},
			ext:    ".md",
			want:   []string{strings.Repeat("あ", 84) + ".md", strings.Repeat("あ", 83) + "-2.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			deduper := NewDeduper()

			for i, title := range tt.titles {
				got := deduper.Unique(title, tt.ext)
				if got != tt.want[i] {
					t.Errorf("Unique(%q) = %q, want %q", title, got, tt.want[i])
				}

				if len(got) > maxNameBytes {
					t.Errorf("Unique(%q) は %d バイトで、上限の %d バイトを超えている", title, len(got), maxNameBytes)
				}
			}
		})
	}
}

func TestDeduperUniqueTwoDigitCounter(t *testing.T) {
	t.Parallel()

	// The byte budget of a name leaves room for the counter, so a counter that grows to two digits
	// shortens the base name by one more byte. An ASCII title makes the difference visible: the
	// base is cut to 250 characters while the counter has one digit, and to 249 once it has two.
	//
	// [Ja] 名前のバイト数の上限は連番の分を空けておくため、連番が 2 桁になるとベース名が 1 バイト
	// 短くなる。ASCII のタイトルはこの差が現れる。連番が 1 桁の間はベース名が 250 文字に切り
	// 詰められ、2 桁になると 249 文字になる。
	deduper := NewDeduper()
	title := strings.Repeat("a", 300)

	want := []string{strings.Repeat("a", 252) + ".md"}
	for counter := 2; counter <= 9; counter++ {
		want = append(want, strings.Repeat("a", 250)+"-"+strconv.Itoa(counter)+".md")
	}

	want = append(want, strings.Repeat("a", 249)+"-10.md")

	for i, w := range want {
		got := deduper.Unique(title, ".md")
		if got != w {
			t.Errorf("Unique() の %d 件目 = %q, want %q", i+1, got, w)
		}

		if len(got) > maxNameBytes {
			t.Errorf("Unique() の %d 件目は %d バイトで、上限の %d バイトを超えている", i+1, len(got), maxNameBytes)
		}
	}
}

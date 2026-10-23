package output

import (
	"testing"
	"unicode/utf8"
)

func TestExcerpt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		body     string
		maxRunes int
		want     string
	}{
		{name: "短い本文はそのまま", body: "本文", maxRunes: 10, want: "本文"},
		{name: "ちょうどの長さは切り詰めない", body: "あいうえお", maxRunes: 5, want: "あいうえお"},
		{name: "長い本文は文字数で切り詰めて省略記号を付ける", body: "あいうえおかきくけこ", maxRunes: 5, want: "あいうえお…"},
		{name: "改行と連続した空白を1つの空白に詰める", body: "# 見出し\n\n本文の\t1行目\r\n  2行目", maxRunes: 100, want: "# 見出し 本文の 1行目 2行目"},
		{name: "全角の空白も詰める", body: "前　　後", maxRunes: 100, want: "前 後"},
		{name: "前後の空白を除く", body: "\n\n  本文  \n", maxRunes: 100, want: "本文"},
		{name: "切り詰めた末尾の空白を残さない", body: "あいう えお", maxRunes: 4, want: "あいう…"},
		{name: "絵文字の途中で切らない", body: "😀😀😀", maxRunes: 2, want: "😀😀…"},
		{name: "空の本文", body: "", maxRunes: 10, want: ""},
		{name: "空白だけの本文", body: " \n\t ", maxRunes: 10, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Excerpt(tt.body, tt.maxRunes)
			if got != tt.want {
				t.Errorf("Excerpt() = %q、期待値 = %q", got, tt.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("Excerpt() = %q は有効なUTF-8ではない", got)
			}
		})
	}
}

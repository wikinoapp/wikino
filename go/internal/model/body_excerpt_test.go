package model_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestBodyExcerpt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "短い本文はそのまま", body: "本文", want: "本文"},
		{name: "ちょうど80文字は切り詰めない", body: strings.Repeat("あ", 80), want: strings.Repeat("あ", 80)},
		{name: "81文字以上は80文字で切り詰めて省略記号を付ける", body: strings.Repeat("あ", 81), want: strings.Repeat("あ", 80) + "…"},
		{name: "改行と連続した空白を1つの空白に詰める", body: "# 見出し\n\n本文の\t1行目\r\n  2行目", want: "# 見出し 本文の 1行目 2行目"},
		{name: "全角の空白も詰める", body: "前　　後", want: "前 後"},
		{name: "前後の空白を除く", body: "\n\n  本文  \n", want: "本文"},
		{name: "切り詰めた末尾の空白を残さない", body: strings.Repeat("あ", 79) + " " + strings.Repeat("い", 10), want: strings.Repeat("あ", 79) + "…"},
		{name: "絵文字の途中で切らない", body: strings.Repeat("😀", 81), want: strings.Repeat("😀", 80) + "…"},
		{name: "空の本文", body: "", want: ""},
		{name: "空白だけの本文", body: " \n\t ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := model.BodyExcerpt(tt.body)
			if got != tt.want {
				t.Errorf("BodyExcerpt() = %q、期待値 = %q", got, tt.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("BodyExcerpt() = %q は有効なUTF-8ではない", got)
			}
		})
	}
}

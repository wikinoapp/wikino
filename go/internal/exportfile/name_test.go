package exportfile

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		title string
		ext   string
		want  string
	}{
		{
			name:  "記号を含まないタイトルはそのまま使う",
			title: "設計メモ",
			ext:   ".md",
			want:  "設計メモ.md",
		},
		{
			name:  "拡張子が空ならディレクトリ名として使える",
			title: "設計メモ",
			ext:   "",
			want:  "設計メモ",
		},
		{
			name:  "OS がファイル名に使えない文字を全角へ置き換える",
			title: `a/b\c:d*e?f"g<h>i|j`,
			ext:   ".md",
			want:  "a／b＼c：d＊e？f”g＜h＞i｜j.md",
		},
		{
			name:  "Obsidian が Wiki リンクの中で構文として読む文字を全角へ置き換える",
			title: "API #1 ^ref [draft]",
			ext:   ".md",
			want:  "API ＃1 ＾ref ［draft］.md",
		},
		{
			name:  "連続するパーセント記号を全角へ置き換える",
			title: "%%コメント%%",
			ext:   ".md",
			want:  "％％コメント％％.md",
		},
		{
			name:  "単独のパーセント記号は置き換えない",
			title: "達成率 100%",
			ext:   ".md",
			want:  "達成率 100%.md",
		},
		{
			name:  "3 つ以上連続するパーセント記号もすべて置き換える",
			title: "%%%記号%",
			ext:   ".md",
			want:  "％％％記号%.md",
		},
		{
			name:  "制御文字を取り除く",
			title: "a\tb\x00c\x7fd",
			ext:   ".md",
			want:  "abcd.md",
		},
		{
			name:  "制御文字を挟んだパーセント記号は連続とみなす",
			title: "%\x00%",
			ext:   ".md",
			want:  "％％.md",
		},
		{
			name:  "書字方向を変える制御文字を取り除く",
			title: "a\u202eb",
			ext:   ".md",
			want:  "ab.md",
		},
		{
			name:  "絵文字を連結する ZWJ は残す",
			title: "👨\u200d👩",
			ext:   ".md",
			want:  "👨\u200d👩.md",
		},
		{
			name:  "不正な UTF-8 のバイト列は置換文字になる",
			title: string([]byte{0xff, 0xfe}) + "a",
			ext:   ".md",
			want:  "\ufffd\ufffda.md",
		},
		{
			name:  "先頭と末尾の空白を取り除く",
			title: "  設計メモ　",
			ext:   ".md",
			want:  "設計メモ.md",
		},
		{
			name:  "先頭と末尾のドットを取り除く",
			title: ".env.",
			ext:   ".md",
			want:  "env.md",
		},
		{
			name:  "空のタイトルは代替名になる",
			title: "",
			ext:   ".md",
			want:  "untitled.md",
		},
		{
			name:  "ドットだけのタイトルは代替名になる",
			title: ".",
			ext:   ".md",
			want:  "untitled.md",
		},
		{
			name:  "ドット 2 つのタイトルは代替名になる",
			title: "..",
			ext:   "",
			want:  "untitled",
		},
		{
			name:  "制御文字だけのタイトルは代替名になる",
			title: "\x00\x01",
			ext:   ".md",
			want:  "untitled.md",
		},
		{
			name:  "Windows の予約デバイス名に接尾辞を付ける",
			title: "CON",
			ext:   ".md",
			want:  "CON_.md",
		},
		{
			name:  "予約デバイス名の判定は大文字小文字を無視する",
			title: "com1",
			ext:   ".md",
			want:  "com1_.md",
		},
		{
			name:  "予約デバイス名で始まるタイトルは最初のドットの前に接尾辞を付ける",
			title: "LPT9.txt",
			ext:   ".md",
			want:  "LPT9_.txt.md",
		},
		{
			name:  "予約デバイス名の末尾に空白があっても接尾辞を付ける",
			title: "CON .txt",
			ext:   ".md",
			want:  "CON _.txt.md",
		},
		{
			name:  "上付き数字の予約デバイス名 COM¹ に接尾辞を付ける",
			title: "COM¹",
			ext:   ".md",
			want:  "COM¹_.md",
		},
		{
			name:  "上付き数字の予約デバイス名 com² は大文字小文字を無視する",
			title: "com²",
			ext:   ".md",
			want:  "com²_.md",
		},
		{
			name:  "上付き数字の予約デバイス名 COM³ は最初のドットの前に接尾辞を付ける",
			title: "COM³.txt",
			ext:   ".md",
			want:  "COM³_.txt.md",
		},
		{
			name:  "拡張子が空でも上付き数字の予約デバイス名 LPT¹ に接尾辞を付ける",
			title: "LPT¹",
			ext:   "",
			want:  "LPT¹_",
		},
		{
			name:  "上付き数字の予約デバイス名 lpt² は大文字小文字を無視する",
			title: "lpt²",
			ext:   ".md",
			want:  "lpt²_.md",
		},
		{
			name:  "上付き数字の予約デバイス名 LPT³ は最初のドットの前に接尾辞を付ける",
			title: "LPT³.txt",
			ext:   ".md",
			want:  "LPT³_.txt.md",
		},
		{
			name:  "予約デバイス名を含むだけのタイトルには接尾辞を付けない",
			title: "CONSOLE",
			ext:   ".md",
			want:  "CONSOLE.md",
		},
		{
			name:  "拡張子を含めて上限バイト数に収める",
			title: strings.Repeat("あ", 100),
			ext:   ".md",
			want:  strings.Repeat("あ", 84) + ".md",
		},
		{
			name:  "上限に収まらない文字は途中で切らずに落とす",
			title: "a" + strings.Repeat("あ", 100),
			ext:   ".md",
			want:  "a" + strings.Repeat("あ", 83) + ".md",
		},
		{
			name:  "上限ちょうどのタイトルは切り詰めない",
			title: strings.Repeat("a", 252),
			ext:   ".md",
			want:  strings.Repeat("a", 252) + ".md",
		},
		{
			name:  "切り詰めで末尾に現れた空白を取り除く",
			title: strings.Repeat("あ", 83) + " " + strings.Repeat("い", 3),
			ext:   ".md",
			want:  strings.Repeat("あ", 83) + ".md",
		},
		{
			name:  "切り詰めで末尾に現れたドットを取り除く",
			title: strings.Repeat("あ", 83) + "." + strings.Repeat("い", 3),
			ext:   ".md",
			want:  strings.Repeat("あ", 83) + ".md",
		},
		{
			name:  "切り詰め後のトリムで予約名になっても接尾辞を付ける",
			title: "CON" + strings.Repeat("　", 83) + "X",
			ext:   ".md",
			want:  "CON_.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Name(tt.title, tt.ext)
			if got != tt.want {
				t.Errorf("Name() = %q, want %q", got, tt.want)
			}

			if len(got) > maxNameBytes {
				t.Errorf("Name() は %d バイトで、上限の %d バイトを超えている", len(got), maxNameBytes)
			}

			if strings.ContainsAny(got, `/\`) {
				t.Errorf("Name() = %q にパス区切りが含まれている", got)
			}

			if !utf8.ValidString(got) {
				t.Errorf("Name() = %q は有効な UTF-8 ではない", got)
			}
		})
	}
}

package ogcard

import (
	"slices"
	"testing"
	"unicode/utf8"

	"golang.org/x/image/math/fixed"
)

// runeCountMeasureは1文字を幅1として数える。折り返し位置を文字数で読めるようにするため
func runeCountMeasure(s string) fixed.Int26_6 {
	return fixed.I(utf8.RuneCountInString(s))
}

func TestWrap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		text     string
		maxWidth int
		maxLines int
		want     []string
	}{
		{
			name:     "和文のみ: 文字間で折り返す",
			text:     "あいうえおかきくけこ",
			maxWidth: 4,
			maxLines: 3,
			want:     []string{"あいうえ", "おかきく", "けこ"},
		},
		{
			name:     "欧文のみ: 単語の境界で折り返し、行末の空白を残さない",
			text:     "hello world foo",
			maxWidth: 11,
			maxLines: 3,
			want:     []string{"hello world", "foo"},
		},
		{
			name:     "和欧混在: 和文と欧文の境界で折り返す",
			text:     "Goで書くOGP画像",
			maxWidth: 5,
			maxLines: 3,
			want:     []string{"Goで書く", "OGP画像"},
		},
		{
			name:     "長い単語: 行に収まらない1語は文字の境界で折り返す",
			text:     "a supercalifragilistic",
			maxWidth: 10,
			maxLines: 3,
			want:     []string{"a", "supercalif", "ragilistic"},
		},
		{
			name:     "行頭禁則: 句点を行頭に置かず、前の文字と一緒に送る",
			text:     "あいう。えお",
			maxWidth: 3,
			maxLines: 3,
			want:     []string{"あい", "う。え", "お"},
		},
		{
			name:     "行頭禁則: 閉じ括弧を行頭に置かない",
			text:     "あいう」えお",
			maxWidth: 3,
			maxLines: 3,
			want:     []string{"あい", "う」え", "お"},
		},
		{
			name:     "行末禁則: 開き括弧を行末に置かない",
			text:     "あい「うえ",
			maxWidth: 3,
			maxLines: 3,
			want:     []string{"あい", "「うえ"},
		},
		{
			name:     "省略: 最大行数を超えたら最終行の末尾を省略記号にする",
			text:     "あいうえおかきくけこさしすせそ",
			maxWidth: 4,
			maxLines: 2,
			want:     []string{"あいうえ", "おかき…"},
		},
		{
			name:     "省略: 欧文の最終行は末尾の空白を除いてから省略記号を付ける",
			text:     "hello world foo bar",
			maxWidth: 11,
			maxLines: 1,
			want:     []string{"hello worl…"},
		},
		{
			name:     "省略: 最終行に余裕があれば削らずに省略記号を付ける",
			text:     "hello world foo",
			maxWidth: 12,
			maxLines: 1,
			want:     []string{"hello world…"},
		},
		{
			name:     "長い単語の途中で最大行数を超えたら省略する",
			text:     "abcdefghijklmnopqrstuvwxyz",
			maxWidth: 10,
			maxLines: 2,
			want:     []string{"abcdefghij", "klmnopqrs…"},
		},
		{
			name:     "最大行数ちょうどに収まるなら省略しない",
			text:     "あいうえおかきく",
			maxWidth: 4,
			maxLines: 2,
			want:     []string{"あいうえ", "おかきく"},
		},
		{
			name:     "空文字列: 行を返さない",
			text:     "",
			maxWidth: 4,
			maxLines: 3,
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := wrap(tt.text, fixed.I(tt.maxWidth), tt.maxLines, runeCountMeasure)
			if !slices.Equal(got, tt.want) {
				t.Errorf("wrap() = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}

func TestSanitize(t *testing.T) {
	t.Parallel()

	// 絵文字だけをフォントに無い文字として扱う
	hasGlyph := func(r rune) bool {
		return r < 0x1F000 && r != 0xFE0F && r != 0x200D
	}

	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "フォントにある文字はそのまま残す",
			text: "Wikinoの使い方 (入門)",
			want: "Wikinoの使い方 (入門)",
		},
		{
			name: "フォントに無い文字を取り除く",
			text: "今日は🍣を食べた",
			want: "今日はを食べた",
		},
		{
			name: "異体字セレクタとZWJを含む絵文字をまとめて取り除く",
			text: "家族👨‍👩‍👧と❤️",
			want: "家族と❤",
		},
		{
			name: "改行とタブを空白にし、連続する空白を1つにまとめる",
			text: "  1行目\n\n2行目\t3行目  ",
			want: "1行目 2行目 3行目",
		},
		{
			name: "取り除いた文字の前後の空白も1つにまとめる",
			text: "前 🎉 後",
			want: "前 後",
		},
		{
			name: "全角空白は文字として残す",
			text: "前　後",
			want: "前　後",
		},
		{
			name: "すべて取り除かれたら空文字列になる",
			text: "🎉🎉",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := sanitize(tt.text, hasGlyph); got != tt.want {
				t.Errorf("sanitize() = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}

func TestHeaderText(t *testing.T) {
	t.Parallel()

	// 絵文字だけをフォントに無い文字として扱う
	hasGlyph := func(r rune) bool {
		return r < 0x1F000
	}

	tests := []struct {
		name string
		card Card
		want string
	}{
		{
			name: "スペース名とトピック名を区切りで連結する",
			card: Card{SpaceName: "スペース", TopicName: "トピック"},
			want: "スペース / トピック",
		},
		{
			name: "スペース名が空になったらトピック名だけにする",
			card: Card{SpaceName: "🎉", TopicName: "トピック"},
			want: "トピック",
		},
		{
			name: "トピック名が空になったらスペース名だけにする",
			card: Card{SpaceName: "スペース", TopicName: "🎉🎉"},
			want: "スペース",
		},
		{
			name: "両方が空になったら空文字列にする",
			card: Card{SpaceName: "🎉", TopicName: "🍣"},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := headerText(tt.card, hasGlyph); got != tt.want {
				t.Errorf("headerText() = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}

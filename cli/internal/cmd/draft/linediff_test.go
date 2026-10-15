package draft

import (
	"strconv"
	"strings"
	"testing"
)

func TestCountLineChanges(t *testing.T) {
	t.Parallel()

	const largeLineCount = 200000
	largeBody := strings.Repeat("x\n", largeLineCount)

	tests := []struct {
		name        string
		old, new    string
		wantAdded   int
		wantDeleted int
	}{
		{name: "同じ本文", old: "a\nb\n", new: "a\nb\n"},
		{name: "どちらも空", old: "", new: ""},
		{name: "空の本文に書く", old: "", new: "a\nb\n", wantAdded: 2},
		{name: "本文を空にする", old: "a\nb\n", new: "", wantDeleted: 2},
		{name: "行を足す", old: "a\nc\n", new: "a\nb\nc\n", wantAdded: 1},
		{name: "行を消す", old: "a\nb\nc\n", new: "a\nc\n", wantDeleted: 1},
		{name: "行を変える", old: "a\nb\nc\n", new: "a\nB\nc\n", wantAdded: 1, wantDeleted: 1},
		{name: "末尾の改行を足す", old: "a\nb", new: "a\nb\n", wantAdded: 1, wantDeleted: 1},
		{name: "行を入れ替える", old: "a\nb\nc\n", new: "c\na\nb\n", wantAdded: 1, wantDeleted: 1},
		{name: "離れた箇所の変更", old: "1\n2\n3\n4\n5\n", new: "0\n1\n2\n4\n5\n6\n", wantAdded: 2, wantDeleted: 1},
		{
			name:      "大きな本文",
			old:       strings.Repeat("x\n", 5000),
			new:       strings.Repeat("x\n", 2500) + "y\n" + strings.Repeat("x\n", 2500),
			wantAdded: 1,
		},
		{name: "大量の追加だけ", new: largeBody, wantAdded: largeLineCount},
		{name: "大量の削除だけ", old: largeBody, wantDeleted: largeLineCount},
		{name: "共通の先頭の後に大量追記", old: "先頭\n", new: "先頭\n" + largeBody, wantAdded: largeLineCount},
		{name: "共通の末尾の前を大量削除", old: largeBody + "末尾\n", new: "末尾\n", wantDeleted: largeLineCount},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			added, deleted := countLineChanges(tt.old, tt.new)
			if added != tt.wantAdded || deleted != tt.wantDeleted {
				t.Errorf("countLineChanges() = +%d -%d、期待値 = +%d -%d", added, deleted, tt.wantAdded, tt.wantDeleted)
			}
		})
	}
}

// BenchmarkCountLineChangesは、追加・削除だけの差分の計算量を行数ごとに比較する。
func BenchmarkCountLineChanges(b *testing.B) {
	for _, n := range []int{10000, 40000, 200000} {
		body := strings.Repeat("x\n", n)
		cases := []struct {
			name     string
			old, new string
		}{
			{name: "追加", new: body},
			{name: "削除", old: body},
			{name: "追記", old: "先頭\n", new: "先頭\n" + body},
		}
		for _, tc := range cases {
			b.Run(tc.name+"/"+strconv.Itoa(n), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					countLineChanges(tc.old, tc.new)
				}
			})
		}
	}
}

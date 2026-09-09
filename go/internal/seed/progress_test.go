package seed

import (
	"bytes"
	"testing"
)

func TestProgress(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	bar := newProgress(&buf, "ページ", 3)
	bar.advance()
	bar.advance()
	bar.finish()

	// Every update rewrites the same line, so the counter stays on one line
	// however many items a step creates.
	//
	// [Ja] 更新のたびに同じ行を書き換えるため、ステップが何件作っても表示は
	// 1 行に収まる。
	want := "\r  ページ 0/3\r  ページ 1/3\r  ページ 2/3\n"
	if got := buf.String(); got != want {
		t.Errorf("進捗の出力が %q であることを期待したが %q だった", want, got)
	}
}

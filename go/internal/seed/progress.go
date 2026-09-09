package seed

import (
	"fmt"
	"io"
)

// progress reports how far a generation step has come. It rewrites a single
// line with a carriage return, so a step that creates hundreds of rows stays
// one line instead of scrolling the terminal.
//
// The counter is written to a plain io.Writer rather than to slog: it is
// throwaway feedback for the developer watching the command, not a record of
// what the run did. What the run did is logged once per step.
//
// out is still expected to be the stream slog writes to, so that the counter
// and the log lines stay in the order they were written.
//
// [Ja] progress は生成ステップの進み具合を表示する。復帰文字で 1 行を書き換える
// ため、数百行を作るステップでも表示は 1 行に収まり、ターミナルが流れない。
//
// カウンタを slog ではなく素の io.Writer へ書くのは、これがコマンドを見ている
// 開発者向けの使い捨てのフィードバックであり、実行内容の記録ではないため。実行
// 内容はステップごとに 1 回ログへ残す。
//
// ただし out には slog の書き込み先と同じストリームを渡す前提とする。カウンタと
// ログ行が書いた順に並ぶようにするため。
type progress struct {
	out     io.Writer
	label   string
	total   int
	current int
}

// newProgress starts a counter labeled label that counts up to total.
//
// [Ja] newProgress は label を見出しにして total までを数えるカウンタを開始する。
func newProgress(out io.Writer, label string, total int) *progress {
	p := &progress{out: out, label: label, total: total}
	p.render()

	return p
}

// advance records that one more item has been created.
//
// [Ja] advance は項目を 1 件作成したことを記録する。
func (p *progress) advance() {
	p.current++
	p.render()
}

// finish closes the line so that the next output starts on its own line.
//
// [Ja] finish は行を閉じ、次の出力が行頭から始まるようにする。
func (p *progress) finish() {
	p.write("\n")
}

// render redraws the counter in place.
//
// [Ja] render はカウンタをその場で描き直す。
func (p *progress) render() {
	p.write(fmt.Sprintf("\r  %s %d/%d", p.label, p.current, p.total))
}

// write sends s to the output, discarding any write error on purpose: the
// output carries progress only, and a run that produced the data must not fail
// because the terminal could not be written to.
//
// [Ja] write は s を出力先へ書き、書き込みエラーは意図的に捨てる。出力先が運ぶのは
// 進捗だけであり、データを投入できた実行がターミナルへ書けなかったことを理由に
// 失敗してはならないため。
func (p *progress) write(s string) {
	_, _ = io.WriteString(p.out, s)
}

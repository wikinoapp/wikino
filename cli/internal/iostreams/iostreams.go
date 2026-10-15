// Package iostreamsは、コマンドが使う標準入出力と、その端末の状態を表す。
//
// コマンドはos.Stdoutや環境変数を直接触らず、このパッケージを経由する。
// テストでは入出力をバッファに差し替え、TTYかどうかも指定して、プロセスを起動せずに
// 人間向けの表示とJSONの出し分けを確かめる。
package iostreams

import (
	"bytes"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// agentEnvKeysは、コーディングエージェントが子プロセスに渡す環境変数。
// エージェントは擬似端末でコマンドを動かすことがあり、TTYの判定だけでは対話できないことを
// 見分けられないため、これらのどれかに値があれば非対話として扱う。
var agentEnvKeys = []string{"CLAUDECODE", "CURSOR_AGENT", "GEMINI_CLI", "AI_AGENT"}

// agentEnvPrefixは、Codexが子プロセスに渡す環境変数の接頭辞 (CODEX_SANDBOXなど)。
const agentEnvPrefix = "CODEX_"

// agentEnvPrefixExcludesは、接頭辞が一致してもエージェントの検出に使わない環境変数。
// CODEX_HOMEは利用者がシェルの設定に書くCodexの設定の場所で、人がシェルから
// wikinoを実行したときにも値があるため除く。
var agentEnvPrefixExcludes = []string{"CODEX_HOME"}

// IOStreamsは標準入力・標準出力・標準エラーと、それぞれがTTYかどうかを持つ。
type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer

	stdinTTY  bool
	stdoutTTY bool
	stderrTTY bool

	agent   bool
	noColor bool
}

// Systemは、プロセスの標準入出力からIOStreamsを作る。
// environは `KEY=value` の形の環境変数の一覧 (os.Environの値)。CODEX_の接頭辞で
// エージェントを検出するため、名前を指定して引くgetenvではなく一覧を受け取る。
func System(environ []string) *IOStreams {
	s := &IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,

		stdinTTY:  isTerminal(os.Stdin),
		stdoutTTY: isTerminal(os.Stdout),
		stderrTTY: isTerminal(os.Stderr),
	}
	s.applyEnv(environ)

	return s
}

// Testは、入出力をバッファにしたIOStreamsを返す。
// 既定ではどれもTTYではなく、エージェントも検出しない。TTYの状態はSet*TTYで変える。
func Test() (*IOStreams, *bytes.Buffer, *bytes.Buffer, *bytes.Buffer) {
	in, out, errOut := &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}

	return &IOStreams{In: in, Out: out, ErrOut: errOut}, in, out, errOut
}

// SetEnvは、environからエージェントとNO_COLORを判定し直す。テストで環境変数の影響を確かめるために使う。
func (s *IOStreams) SetEnv(environ []string) {
	s.applyEnv(environ)
}

// SetStdinTTYは標準入力がTTYかどうかを設定する。
func (s *IOStreams) SetStdinTTY(isTTY bool) {
	s.stdinTTY = isTTY
}

// SetStdoutTTYは標準出力がTTYかどうかを設定する。
func (s *IOStreams) SetStdoutTTY(isTTY bool) {
	s.stdoutTTY = isTTY
}

// SetStderrTTYは標準エラーがTTYかどうかを設定する。
func (s *IOStreams) SetStderrTTY(isTTY bool) {
	s.stderrTTY = isTTY
}

// IsStdoutTTYは標準出力がTTYかどうかを返す。人間向けの表示とJSONの出し分けに使う。
func (s *IOStreams) IsStdoutTTY() bool {
	return s.stdoutTTY
}

// IsStderrTTYは標準エラーがTTYかどうかを返す。
func (s *IOStreams) IsStderrTTY() bool {
	return s.stderrTTY
}

// CanPromptは、利用者に対話で入力を求めてよいかを返す。
// 標準入力と標準出力がどちらもTTYで、エージェントの環境変数も無いときだけ対話できるとする。
// 対話できないときにプロンプトを出すと、エージェントやCIは入力を待ったまま止まるためである。
func (s *IOStreams) CanPrompt() bool {
	return s.stdinTTY && s.stdoutTTY && !s.agent
}

// ColorEnabledは標準出力に色を付けてよいかを返す。
// TTYのときだけ付け、NO_COLOR (https://no-color.org/) に値があれば付けない。
func (s *IOStreams) ColorEnabled() bool {
	return s.stdoutTTY && !s.noColor
}

func (s *IOStreams) applyEnv(environ []string) {
	s.agent = false
	s.noColor = false

	for _, entry := range environ {
		key, value, _ := strings.Cut(entry, "=")
		if value == "" {
			continue
		}

		if key == "NO_COLOR" {
			s.noColor = true
		}
		if isAgentEnvKey(key) {
			s.agent = true
		}
	}
}

func isAgentEnvKey(key string) bool {
	for _, k := range agentEnvKeys {
		if key == k {
			return true
		}
	}

	if !strings.HasPrefix(key, agentEnvPrefix) {
		return false
	}
	for _, k := range agentEnvPrefixExcludes {
		if key == k {
			return false
		}
	}

	return true
}

func isTerminal(f *os.File) bool {
	// ファイルディスクリプタはuintptrで、intに収まらない値は端末ではありえない。
	fd := f.Fd()
	if fd > uintptr(^uint(0)>>1) {
		return false
	}

	return term.IsTerminal(int(fd))
}

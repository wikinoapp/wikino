package iostreams

import "testing"

func TestCanPrompt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		stdinTTY  bool
		stdoutTTY bool
		environ   []string
		want      bool
	}{
		{name: "標準入力と標準出力がTTY", stdinTTY: true, stdoutTTY: true, want: true},
		{name: "標準入力がTTYでない", stdinTTY: false, stdoutTTY: true, want: false},
		{name: "標準出力がTTYでない", stdinTTY: true, stdoutTTY: false, want: false},
		{name: "Claude Code", stdinTTY: true, stdoutTTY: true, environ: []string{"CLAUDECODE=1"}, want: false},
		{name: "Codex", stdinTTY: true, stdoutTTY: true, environ: []string{"CODEX_SANDBOX=seatbelt"}, want: false},
		{name: "Cursor", stdinTTY: true, stdoutTTY: true, environ: []string{"CURSOR_AGENT=1"}, want: false},
		{name: "Gemini CLI", stdinTTY: true, stdoutTTY: true, environ: []string{"GEMINI_CLI=1"}, want: false},
		{name: "AI_AGENT", stdinTTY: true, stdoutTTY: true, environ: []string{"AI_AGENT=my-agent"}, want: false},
		{name: "CODEX_HOMEは利用者の設定なのでエージェントとみなさない", stdinTTY: true, stdoutTTY: true, environ: []string{"CODEX_HOME=/home/me/.codex"}, want: true},
		{name: "値が空の環境変数はエージェントとみなさない", stdinTTY: true, stdoutTTY: true, environ: []string{"CLAUDECODE="}, want: true},
		{name: "関係の無い環境変数", stdinTTY: true, stdoutTTY: true, environ: []string{"HOME=/home/me", "CODEXX=1"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ios, _, _, _ := Test()
			ios.SetStdinTTY(tt.stdinTTY)
			ios.SetStdoutTTY(tt.stdoutTTY)
			ios.SetEnv(tt.environ)

			if got := ios.CanPrompt(); got != tt.want {
				t.Errorf("CanPrompt() = %v、期待値 = %v", got, tt.want)
			}
		})
	}
}

func TestColorEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		stdoutTTY bool
		environ   []string
		want      bool
	}{
		{name: "TTY", stdoutTTY: true, want: true},
		{name: "TTYでない", stdoutTTY: false, want: false},
		{name: "NO_COLOR", stdoutTTY: true, environ: []string{"NO_COLOR=1"}, want: false},
		{name: "値が空のNO_COLORは無視する", stdoutTTY: true, environ: []string{"NO_COLOR="}, want: true},
		{name: "エージェントでもTTYなら色を付ける", stdoutTTY: true, environ: []string{"CLAUDECODE=1"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ios, _, _, _ := Test()
			ios.SetStdoutTTY(tt.stdoutTTY)
			ios.SetEnv(tt.environ)

			if got := ios.ColorEnabled(); got != tt.want {
				t.Errorf("ColorEnabled() = %v、期待値 = %v", got, tt.want)
			}
		})
	}
}

func TestTestDefaults(t *testing.T) {
	t.Parallel()

	ios, _, _, _ := Test()
	if ios.IsStdoutTTY() || ios.IsStderrTTY() || ios.CanPrompt() || ios.ColorEnabled() {
		t.Error("テスト用のIOStreamsは、既定でTTYでも対話可能でもないことを期待した")
	}
}

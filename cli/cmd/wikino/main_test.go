package main

import (
	"io"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/iostreams"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		env        map[string]string
		stderrTTY  bool
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{
			name:       "--versionはバージョンを出して成功する",
			args:       []string{"--version"},
			wantCode:   output.ExitOK,
			wantStdout: "wikino version 1.2.3\n",
		},
		{
			name:       "未知のサブコマンドは候補を出さずに使い方の誤りで失敗する",
			args:       []string{"nosuchcommand"},
			env:        map[string]string{"LANG": "ja_JP.UTF-8"},
			stderrTTY:  true,
			wantCode:   output.ExitUsage,
			wantStderr: "不明なサブコマンドです: \"nosuchcommand\"\n`wikino --help` で使い方を確認してください。\n",
		},
		{
			name:       "未知のサブコマンドの後の--versionでも失敗する",
			args:       []string{"nosuchcommand", "--version"},
			env:        map[string]string{"WIKINO_LANG": "ja"},
			stderrTTY:  true,
			wantCode:   output.ExitUsage,
			wantStderr: "不明なサブコマンドです: \"nosuchcommand\"\n`wikino --help` で使い方を確認してください。\n",
		},
		{
			name:       "--versionの後の未知のサブコマンドでも失敗する",
			args:       []string{"--version", "nosuchcommand"},
			env:        map[string]string{"WIKINO_LANG": "ja"},
			stderrTTY:  true,
			wantCode:   output.ExitUsage,
			wantStderr: "不明なサブコマンドです: \"nosuchcommand\"\n`wikino --help` で使い方を確認してください。\n",
		},
		{
			name:       "サブコマンドが無いときは使い方の誤りで失敗する",
			args:       []string{},
			env:        map[string]string{"LANG": "ja_JP.UTF-8"},
			stderrTTY:  true,
			wantCode:   output.ExitUsage,
			wantStderr: "サブコマンドを指定してください。\n`wikino --help` で使い方を確認してください。\n",
		},
		{
			name:       "日本語以外のロケールでは英語で出す",
			args:       []string{"nosuchcommand"},
			env:        map[string]string{"LANG": "en_US.UTF-8"},
			stderrTTY:  true,
			wantCode:   output.ExitUsage,
			wantStderr: "Unknown subcommand: \"nosuchcommand\"\nRun `wikino --help` to see the usage.\n",
		},
		{
			name:       "標準エラーがTTYでなければエラーをJSONで出す",
			args:       []string{"nosuchcommand"},
			env:        map[string]string{"WIKINO_LANG": "ja"},
			wantCode:   output.ExitUsage,
			wantStderr: `{"error":{"code":"usage_error","message":"不明なサブコマンドです: \"nosuchcommand\"","hint":"` + "`wikino --help` で使い方を確認してください。" + `"}}` + "\n",
		},
		{
			name:       "標準エラーがTTYでも--jsonならエラーをJSONで出す",
			args:       []string{"--json", "nosuchcommand"},
			env:        map[string]string{"WIKINO_LANG": "en"},
			stderrTTY:  true,
			wantCode:   output.ExitUsage,
			wantStderr: `{"error":{"code":"usage_error","message":"Unknown subcommand: \"nosuchcommand\"","hint":"Run ` + "`wikino --help`" + ` to see the usage."}}` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ios, _, stdout, stderr := iostreams.Test()
			ios.SetStderrTTY(tt.stderrTTY)

			code := run(tt.args, "1.2.3", ios, func(key string) string { return tt.env[key] })

			if code != tt.wantCode {
				t.Errorf("終了コード = %d、期待値 = %d", code, tt.wantCode)
			}
			if stdout.String() != tt.wantStdout {
				t.Errorf("標準出力 = %q、期待値 = %q", stdout.String(), tt.wantStdout)
			}
			if stderr.String() != tt.wantStderr {
				t.Errorf("標準エラー = %q、期待値 = %q", stderr.String(), tt.wantStderr)
			}
			if strings.Contains(stderr.String(), "Did you mean") {
				t.Errorf("候補を出さないことを期待したが、標準エラーに含まれていた: %q", stderr.String())
			}
		})
	}
}

func TestResolveVersion(t *testing.T) {
	t.Parallel()

	moduleInfo := func(version string) *debug.BuildInfo {
		return &debug.BuildInfo{Main: debug.Module{Version: version}}
	}

	tests := []struct {
		name           string
		ldflagsVersion string
		info           *debug.BuildInfo
		ok             bool
		want           string
	}{
		{name: "ldflagsの値を優先する", ldflagsVersion: "1.2.3", info: moduleInfo("v1.0.0"), ok: true, want: "1.2.3"},
		{name: "go installではモジュールのバージョンを使う", info: moduleInfo("v1.0.0"), ok: true, want: "v1.0.0"},
		{name: "手元のビルドはdev", info: moduleInfo("(devel)"), ok: true, want: "dev"},
		{name: "ビルド情報が無ければdev", info: nil, ok: false, want: "dev"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := resolveVersion(tt.ldflagsVersion, tt.info, tt.ok)
			if got != tt.want {
				t.Errorf("resolveVersion() = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}

// failedWriterは、書き込み先を閉じた場合の出力エラーを再現する。
type failedWriter struct{}

func (failedWriter) Write([]byte) (int, error) {
	return 0, io.ErrClosedPipe
}

func TestRunOutputFailure(t *testing.T) {
	t.Parallel()

	for _, flag := range []string{"--help", "--version"} {
		t.Run(flag, func(t *testing.T) {
			t.Parallel()

			ios, _, _, stderr := iostreams.Test()
			ios.Out = failedWriter{}
			ios.SetStderrTTY(true)

			code := run([]string{flag}, "1.2.3", ios, func(string) string { return "ja" })
			if code != output.ExitError {
				t.Errorf("終了コード = %d、期待値 = %d", code, output.ExitError)
			}
			if got, want := stderr.String(), io.ErrClosedPipe.Error()+"\n"; got != want {
				t.Errorf("標準エラー = %q、期待値 = %q", got, want)
			}
		})
	}
}

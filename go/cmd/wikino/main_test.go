package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	t.Parallel()

	// serveサブコマンドはここでは扱わない。ポートを占有しシャットダウンまで
	// ブロックするため、その確認は振り分けのテストではなくE2Eスイートの担当。
	tests := []struct {
		name string
		args []string
	}{
		{name: "引数無しではusageを表示して非ゼロ終了する", args: []string{}},
		{name: "未知のサブコマンドではusageを表示して非ゼロ終了する", args: []string{"nosuchcommand"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr bytes.Buffer

			code := run(tt.args, &stdout, &stderr)

			if code != exitUsage {
				t.Errorf("終了コードが%dであることを期待したが%dだった", exitUsage, code)
			}
			if !strings.Contains(stderr.String(), "使い方: wikino <command>") {
				t.Errorf("usageの出力を期待したが次の内容だった: %q", stderr.String())
			}
		})
	}
}

func TestRunUnknownSubcommandNamesTheSubcommand(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer

	run([]string{"nosuchcommand"}, &stdout, &stderr)

	if !strings.Contains(stderr.String(), `不明なサブコマンドです: "nosuchcommand"`) {
		t.Errorf("未知のサブコマンド名を含む出力を期待したが次の内容だった: %q", stderr.String())
	}
}

// TestRunDevCredentialsDispatchPassesTheRoleArgumentsは、振り分けとdevcreds
// サブコマンドの間の配線を確認する。サブコマンドへ届くのは、コマンドライン全体では
// なく、サブコマンド名の後ろに続く引数である必要がある。
//
// コマンドライン全体を渡すと、サブコマンド名そのものが役割に見える。それは
// サブコマンドが受理する唯一の引数の個数でもあるため、その実行はusageを報告せず、
// "devcreds" という名前の役割を引きに行くことになる。しかも本パッケージの他の
// テストはすべて通ったままになる。
//
// 成功時の引きはここでは扱わない。名簿を読むが、それは本パッケージではなくモジュール
// ルートが持つファイルであるため。引数を受け取った後のdevcredsの振る舞いは
// devcreds_test.goが扱う。
func TestRunDevCredentialsDispatchPassesTheRoleArguments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "役割を指定していないとき", args: []string{"devcreds"}},
		{name: "役割を2つ以上指定したとき", args: []string{"devcreds", "owner", "collaborator"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr bytes.Buffer

			code := run(tt.args, &stdout, &stderr)

			if code != exitUsage {
				t.Errorf("終了コードが%dであることを期待したが%dだった", exitUsage, code)
			}
			if !strings.Contains(stderr.String(), "使い方: wikino devcreds <role>") {
				t.Errorf("devcredsのusageの出力を期待したが次の内容だった: %q", stderr.String())
			}
			// シェルはこのストリームを資格情報そのものとして読むため、答えられ
			// ない実行では、振り分けがdevcredsへ渡したストリームが汚れないままで
			// ある必要がある。
			if stdout.Len() != 0 {
				t.Errorf("標準出力が空であることを期待したが%qだった", stdout.String())
			}
		})
	}
}

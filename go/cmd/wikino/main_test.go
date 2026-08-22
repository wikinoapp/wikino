package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	t.Parallel()

	// The serve subcommand is not covered here: it binds a port and blocks
	// until shutdown, so exercising it belongs to the E2E suite rather than to
	// a dispatch test.
	//
	// [Ja] serve サブコマンドはここでは扱わない。ポートを占有しシャットダウンまで
	// ブロックするため、その確認は振り分けのテストではなく E2E スイートの担当。
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
				t.Errorf("終了コードが %d であることを期待したが %d だった", exitUsage, code)
			}
			if !strings.Contains(stderr.String(), "usage: wikino <command>") {
				t.Errorf("usageの出力を期待したが次の内容だった: %q", stderr.String())
			}
		})
	}
}

func TestRunUnknownSubcommandNamesTheSubcommand(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer

	run([]string{"nosuchcommand"}, &stdout, &stderr)

	if !strings.Contains(stderr.String(), `unknown subcommand: "nosuchcommand"`) {
		t.Errorf("未知のサブコマンド名を含む出力を期待したが次の内容だった: %q", stderr.String())
	}
}

// TestRunDevCredentialsDispatchPassesTheRoleArguments covers the wiring between
// the dispatch and the devcreds subcommand: what reaches it has to be the
// arguments that follow the subcommand name, not the whole command line.
//
// Handing it the whole command line would make the subcommand name itself look
// like the role, which is the one argument count the subcommand accepts. The
// invocation would then go on to look a role named "devcreds" up rather than
// reporting the usage, and every other test in this package would still pass.
//
// The successful lookup is not reached from here: it reads the roster, a file
// the module root holds rather than this package. What devcreds does once it
// has its arguments is covered in devcreds_test.go.
//
// [Ja] TestRunDevCredentialsDispatchPassesTheRoleArguments は、振り分けと devcreds
// サブコマンドの間の配線を確認する。サブコマンドへ届くのは、コマンドライン全体では
// なく、サブコマンド名の後ろに続く引数である必要がある。
//
// コマンドライン全体を渡すと、サブコマンド名そのものが役割に見える。それは
// サブコマンドが受理する唯一の引数の個数でもあるため、その実行は usage を報告せず、
// "devcreds" という名前の役割を引きに行くことになる。しかも本パッケージの他の
// テストはすべて通ったままになる。
//
// 成功時の引きはここでは扱わない。名簿を読むが、それは本パッケージではなくモジュール
// ルートが持つファイルであるため。引数を受け取った後の devcreds の振る舞いは
// devcreds_test.go が扱う。
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
				t.Errorf("終了コードが %d であることを期待したが %d だった", exitUsage, code)
			}
			if !strings.Contains(stderr.String(), "usage: wikino devcreds <role>") {
				t.Errorf("devcredsのusageの出力を期待したが次の内容だった: %q", stderr.String())
			}
			// The shell reads this stream as the credentials themselves, so
			// the dispatch has to hand devcreds a stream it leaves untouched
			// when it cannot answer.
			//
			// [Ja] シェルはこのストリームを資格情報そのものとして読むため、答えられ
			// ない実行では、振り分けが devcreds へ渡したストリームが汚れないままで
			// ある必要がある。
			if stdout.Len() != 0 {
				t.Errorf("標準出力が空であることを期待したが %q だった", stdout.String())
			}
		})
	}
}

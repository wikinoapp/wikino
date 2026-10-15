package root

import (
	"reflect"
	"testing"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/cmdtest"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

func TestHelpJSON(t *testing.T) {
	t.Parallel()

	got := executeWith(t, i18n.LangJa, []string{"sample", "get", "--help", "--json"}, func(root *cobra.Command, loc *i18n.Localizer) {
		addSampleGroup(root, loc)
		get, _, _ := root.Find([]string{"sample", "get"})
		get.Long = "サンプルを番号で取得します。"
		get.Flags().StringP("output", "O", "", "保存先の`path`")
	})
	if got.err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", got.err)
	}

	var help helpJSON
	cmdtest.Decode(t, got.stdout, &help)

	if help.Command != "wikino sample get" || help.Usage != "wikino sample get <n> [flags]" {
		t.Errorf("コマンド = %q・使い方 = %q", help.Command, help.Usage)
	}
	if help.Short != "サンプルを取得する" || help.Long != "サンプルを番号で取得します。" {
		t.Errorf("短い説明 = %q・説明 = %q", help.Short, help.Long)
	}
	if want := []helpArg{{Name: "n", Required: true}}; !reflect.DeepEqual(help.Args, want) {
		t.Errorf("引数 = %+v、期待値 = %+v", help.Args, want)
	}
	if help.Examples != "  wikino sample get 12" || help.Output != "サンプルのJSON" || help.Errors != "終了コード5: サンプルが無い" {
		t.Errorf("例 = %q・出力 = %q・エラー = %q", help.Examples, help.Output, help.Errors)
	}
	if len(help.Subcommands) != 0 {
		t.Errorf("サブコマンドが無いことを期待したが %+v だった", help.Subcommands)
	}

	wantFlags := []helpFlag{{Name: "output", Shorthand: "O", Type: "string", Default: "", Description: "保存先のpath"}}
	if !reflect.DeepEqual(help.Flags, wantFlags) {
		t.Errorf("フラグ = %+v、期待値 = %+v", help.Flags, wantFlags)
	}

	// 継承したフラグには、ルートのコマンドの共通のフラグが入る。
	inherited := map[string]helpFlag{}
	for _, f := range help.InheritedFlags {
		inherited[f.Name] = f
	}
	for _, name := range []string{"help", "json", "fields", "jq", "host", "space", "member"} {
		if _, ok := inherited[name]; !ok {
			t.Errorf("継承したフラグに %s が無い: %+v", name, help.InheritedFlags)
		}
	}
	if f := inherited["help"]; f.Shorthand != "h" || f.Type != "bool" || f.Default != "false" {
		t.Errorf("--helpの項目 = %+v", f)
	}
	if f := inherited["fields"]; f.Description != "出す項目をカンマ区切りで選ぶ (例: number,title。一覧では各要素に適用する)" {
		t.Errorf("--fieldsの説明からバッククォートを取り除くことを期待したが %q だった", f.Description)
	}
	if _, ok := inherited["version"]; ok {
		t.Error("ルートのコマンドだけのフラグ (--version) が継承したフラグに入っている")
	}
}

func TestHelpJSONGroup(t *testing.T) {
	t.Parallel()

	got := executeWith(t, i18n.LangEn, []string{"sample", "--help", "--json"}, addSampleGroup)
	if got.err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", got.err)
	}

	var help helpJSON
	cmdtest.Decode(t, got.stdout, &help)

	if help.Usage != "wikino sample <command>" {
		t.Errorf("使い方 = %q、期待値 = %q", help.Usage, "wikino sample <command>")
	}
	want := []helpSubcommand{{Name: "get", Command: "wikino sample get", Short: "サンプルを取得する"}}
	if !reflect.DeepEqual(help.Subcommands, want) {
		t.Errorf("サブコマンド = %+v、期待値 = %+v", help.Subcommands, want)
	}
	if len(help.Args) != 0 || len(help.Flags) != 0 {
		t.Errorf("引数 = %+v・フラグ = %+v、どちらも空を期待した", help.Args, help.Flags)
	}
}

func TestHelpJSONOutputFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "--fieldsで項目を選ぶ",
			args: []string{"sample", "get", "--help", "--fields", "command,args"},
			want: `{"args":[{"name":"n","required":true,"variadic":false}],"command":"wikino sample get"}` + "\n",
		},
		{
			name: "--jqで絞り込む",
			args: []string{"sample", "--help", "--jq", ".subcommands[].command"},
			want: "wikino sample get\n",
		},
		{
			name: "--jsonが無ければ非TTYでも人間向けのヘルプを出す",
			args: []string{"sample", "get", "--help"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := executeWith(t, i18n.LangJa, tt.args, addSampleGroup)
			if got.err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", got.err)
			}
			if tt.want == "" {
				if got.stdout == "" || got.stdout[0] == '{' {
					t.Errorf("人間向けのヘルプを期待したが %q だった", got.stdout)
				}
				return
			}
			if got.stdout != tt.want {
				t.Errorf("標準出力 = %q、期待値 = %q", got.stdout, tt.want)
			}
		})
	}
}

func TestHelpJSONErrors(t *testing.T) {
	t.Parallel()

	t.Run("--fieldsの未知の項目", func(t *testing.T) {
		t.Parallel()

		got := executeWith(t, i18n.LangJa, []string{"sample", "get", "--help", "--fields", "nosuchfield"}, addSampleGroup)

		cliErr := cmdtest.AssertCLIError(t, got.err, output.CodeUsage, output.ExitUsage)
		if cliErr.Message != `不明な項目です: "nosuchfield"` {
			t.Errorf("メッセージ = %q", cliErr.Message)
		}
		if got.stdout != "" {
			t.Errorf("何も出力しないことを期待したが %q だった", got.stdout)
		}
	})

	t.Run("グループの未知のサブコマンド", func(t *testing.T) {
		t.Parallel()

		got := executeWith(t, i18n.LangJa, []string{"sample", "nosuchcommand", "--help", "--json"}, addSampleGroup)

		assertUsageError(t, got.err, `不明なサブコマンドです: "nosuchcommand"`, "`wikino sample --help` で使い方を確認してください。")
		if got.stdout != "" {
			t.Errorf("何も出力しないことを期待したが %q だった", got.stdout)
		}
	})
}

func TestParseArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		use  string
		want []helpArg
	}{
		{use: "list", want: []helpArg{}},
		{use: "get <page_number>", want: []helpArg{{Name: "page_number", Required: true}}},
		{use: "login [<space>]", want: []helpArg{{Name: "space"}}},
		{use: "search <query>...", want: []helpArg{{Name: "query", Required: true, Variadic: true}}},
		{
			use:  "get <page_number> <revision_id>",
			want: []helpArg{{Name: "page_number", Required: true}, {Name: "revision_id", Required: true}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.use, func(t *testing.T) {
			t.Parallel()

			if got := parseArgs(tt.use); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseArgs(%q) = %+v、期待値 = %+v", tt.use, got, tt.want)
			}
		})
	}
}

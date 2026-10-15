package output

import (
	"bytes"
	"errors"
	"testing"

	"github.com/spf13/pflag"

	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/iostreams"
)

// testPageはAPIの応答の構造体の代わり。
type testPage struct {
	Number int64   `json:"number"`
	Title  *string `json:"title"`
	Body   string  `json:"body"`
	URL    string  `json:"url"`
}

type testPageList struct {
	Items      []testPage `json:"items"`
	NextCursor *string    `json:"next_cursor"`
}

var pageSpec = Spec{
	Fields:  []string{"number", "title", "body", "url"},
	Default: []string{"number", "title", "url"},
}

func ptr[T any](v T) *T {
	return &v
}

type printerSetup struct {
	args      []string
	stdoutTTY bool
	environ   []string
	spec      Spec
}

func newTestPrinter(t *testing.T, s printerSetup) (*Printer, *bytes.Buffer, error) {
	t.Helper()

	loc := i18n.NewLocalizer(i18n.LangJa)
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	AddFlags(fs, loc)
	if err := fs.Parse(s.args); err != nil {
		t.Fatalf("フラグを解析できません: %v", err)
	}

	ios, _, stdout, _ := iostreams.Test()
	ios.SetStdoutTTY(s.stdoutTTY)
	ios.SetEnv(s.environ)

	p, err := NewPrinter(ios, loc, fs, s.spec)

	return p, stdout, err
}

func TestPrinterIsJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup printerSetup
		want  bool
	}{
		{name: "TTYなら人間向け", setup: printerSetup{stdoutTTY: true}, want: false},
		{name: "TTYでなければJSON", setup: printerSetup{stdoutTTY: false}, want: true},
		{name: "TTYでも--jsonならJSON", setup: printerSetup{args: []string{"--json"}, stdoutTTY: true}, want: true},
		{name: "TTYでも--fieldsならJSON", setup: printerSetup{args: []string{"--fields", "title"}, stdoutTTY: true, spec: pageSpec}, want: true},
		{name: "TTYでも--jqならJSON", setup: printerSetup{args: []string{"--jq", ".title"}, stdoutTTY: true}, want: true},
		{name: "エージェントの環境変数は表示の形を変えない", setup: printerSetup{stdoutTTY: true, environ: []string{"CLAUDECODE=1"}}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p, _, err := newTestPrinter(t, tt.setup)
			if err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}
			if got := p.IsJSON(); got != tt.want {
				t.Errorf("IsJSON() = %v、期待値 = %v", got, tt.want)
			}
		})
	}
}

func TestPrinterPrintItem(t *testing.T) {
	t.Parallel()

	page := testPage{Number: 12, Title: ptr("議事録 <草案> & メモ"), Body: "本文\n", URL: "https://example.com/s/example/pages/12"}

	tests := []struct {
		name  string
		setup printerSetup
		want  string
	}{
		{
			name:  "非TTYでは1行のJSONにし、HTMLの記号をエスケープしない",
			setup: printerSetup{spec: Spec{Fields: pageSpec.Fields}},
			want:  `{"body":"本文\n","number":12,"title":"議事録 <草案> & メモ","url":"https://example.com/s/example/pages/12"}` + "\n",
		},
		{
			name:  "TTYで--jsonなら字下げする",
			setup: printerSetup{args: []string{"--json"}, stdoutTTY: true, spec: Spec{Fields: pageSpec.Fields}},
			want:  "{\n  \"body\": \"本文\\n\",\n  \"number\": 12,\n  \"title\": \"議事録 <草案> & メモ\",\n  \"url\": \"https://example.com/s/example/pages/12\"\n}\n",
		},
		{
			name:  "既定の項目を出す",
			setup: printerSetup{spec: pageSpec},
			want:  `{"number":12,"title":"議事録 <草案> & メモ","url":"https://example.com/s/example/pages/12"}` + "\n",
		},
		{
			name:  "--fieldsで選んだ項目だけを出す",
			setup: printerSetup{args: []string{"--fields", "number,body"}, spec: pageSpec},
			want:  `{"body":"本文\n","number":12}` + "\n",
		},
		{
			name:  "--fieldsを繰り返して指定できる",
			setup: printerSetup{args: []string{"--fields", "number", "--fields", "url"}, spec: pageSpec},
			want:  `{"number":12,"url":"https://example.com/s/example/pages/12"}` + "\n",
		},
		{
			name:  "--jqの文字列の結果は引用符を付けずに出す",
			setup: printerSetup{args: []string{"--jq", ".body"}, spec: Spec{Fields: pageSpec.Fields}},
			want:  "本文\n\n",
		},
		{
			name:  "--jqは--fieldsの後に適用する",
			setup: printerSetup{args: []string{"--fields", "number,title", "--jq", "keys"}, spec: pageSpec},
			want:  `["number","title"]` + "\n",
		},
		{
			name:  "--jqの複数の結果を1つずつ出す",
			setup: printerSetup{args: []string{"--jq", ".number, .title, null"}, spec: pageSpec},
			want:  "12\n議事録 <草案> & メモ\nnull\n",
		},
		{
			name:  "--jqのemptyは何も出さない",
			setup: printerSetup{args: []string{"--jq", "empty"}, spec: pageSpec},
			want:  "",
		},
		{
			name:  "--jqのhaltで出力を止める",
			setup: printerSetup{args: []string{"--jq", ".number, halt, .title"}, spec: pageSpec},
			want:  "12\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p, stdout, err := newTestPrinter(t, tt.setup)
			if err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}
			if err := p.PrintItem(page); err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}
			if got := stdout.String(); got != tt.want {
				t.Errorf("標準出力 = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}

func TestPrinterPrintList(t *testing.T) {
	t.Parallel()

	list := testPageList{
		Items: []testPage{
			{Number: 1, Title: ptr("A"), Body: "本文A", URL: "https://example.com/1"},
			{Number: 2, Title: nil, Body: "本文B", URL: "https://example.com/2"},
		},
		NextCursor: ptr("abc"),
	}

	tests := []struct {
		name  string
		list  testPageList
		setup printerSetup
		want  string
	}{
		{
			name:  "既定では各要素に既定の項目を適用し、本文を出さない",
			list:  list,
			setup: printerSetup{spec: pageSpec},
			want:  `{"items":[{"number":1,"title":"A","url":"https://example.com/1"},{"number":2,"title":null,"url":"https://example.com/2"}],"next_cursor":"abc"}` + "\n",
		},
		{
			name:  "--fieldsにbodyを含めたときだけ本文を出す",
			list:  list,
			setup: printerSetup{args: []string{"--fields", "number,body"}, spec: pageSpec},
			want:  `{"items":[{"body":"本文A","number":1},{"body":"本文B","number":2}],"next_cursor":"abc"}` + "\n",
		},
		{
			name:  "--jqは一覧全体に適用する",
			list:  list,
			setup: printerSetup{args: []string{"--jq", ".items[].number"}, spec: pageSpec},
			want:  "1\n2\n",
		},
		{
			name:  "0件でも空の一覧を出す",
			list:  testPageList{Items: []testPage{}},
			setup: printerSetup{spec: pageSpec},
			want:  `{"items":[],"next_cursor":null}` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p, stdout, err := newTestPrinter(t, tt.setup)
			if err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}
			if err := p.PrintList(tt.list); err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}
			if got := stdout.String(); got != tt.want {
				t.Errorf("標準出力 = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}

func TestPrinterPrintLine(t *testing.T) {
	t.Parallel()

	pages := []testPage{
		{Number: 1, Title: ptr("A"), Body: "本文A\n2行目", URL: "https://example.com/1"},
		{Number: 9007199254740993, Title: ptr("B"), Body: "本文B", URL: "https://example.com/2"},
	}

	tests := []struct {
		name  string
		setup printerSetup
		want  string
	}{
		{
			name:  "1件を1行のJSONで出し、TTYでも字下げしない",
			setup: printerSetup{args: []string{"--json"}, stdoutTTY: true, spec: pageSpec},
			want: `{"number":1,"title":"A","url":"https://example.com/1"}` + "\n" +
				`{"number":9007199254740993,"title":"B","url":"https://example.com/2"}` + "\n",
		},
		{
			name:  "--fieldsと--jqは1件ごとに適用する",
			setup: printerSetup{args: []string{"--fields", "number,title", "--jq", ".title"}, spec: pageSpec},
			want:  "A\nB\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p, stdout, err := newTestPrinter(t, tt.setup)
			if err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}
			for _, page := range pages {
				if err := p.PrintLine(page); err != nil {
					t.Fatalf("エラーを期待しなかったが %v だった", err)
				}
			}
			if got := stdout.String(); got != tt.want {
				t.Errorf("標準出力 = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}

func TestNewPrinterUsageErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		wantMessage string
		wantHint    string
	}{
		{
			name:        "未知の項目名",
			args:        []string{"--fields", "number,titel"},
			wantMessage: `不明な項目です: "titel"`,
			wantHint:    "選べる項目: number,title,body,url",
		},
		{
			name:        "jqの構文の誤り",
			args:        []string{"--jq", ".title |"},
			wantMessage: "--jqの式を解釈できません: unexpected EOF",
			wantHint:    "式を直してから実行し直してください。要求はまだ送っていません。",
		},
		{
			name:        "jqの未定義の関数",
			args:        []string{"--jq", "nosuchfunc"},
			wantMessage: "--jqの式を解釈できません: function not defined: nosuchfunc/0",
			wantHint:    "式を直してから実行し直してください。要求はまだ送っていません。",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := newTestPrinter(t, printerSetup{args: tt.args, spec: pageSpec})

			var cliErr *Error
			if !errors.As(err, &cliErr) {
				t.Fatalf("使い方の誤りを期待したが %v (%T) だった", err, err)
			}
			if cliErr.Code != CodeUsage || cliErr.ExitCode != ExitUsage {
				t.Errorf("コード = %q・終了コード = %d、期待値 = %q・%d", cliErr.Code, cliErr.ExitCode, CodeUsage, ExitUsage)
			}
			if cliErr.Message != tt.wantMessage {
				t.Errorf("メッセージ = %q、期待値 = %q", cliErr.Message, tt.wantMessage)
			}
			if cliErr.Hint != tt.wantHint {
				t.Errorf("ヒント = %q、期待値 = %q", cliErr.Hint, tt.wantHint)
			}
		})
	}
}

func TestPrinterJQRuntimeError(t *testing.T) {
	t.Parallel()

	p, stdout, err := newTestPrinter(t, printerSetup{args: []string{"--jq", ".number, (.title | tonumber)"}, spec: pageSpec})
	if err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	err = p.PrintItem(testPage{Number: 12, Title: ptr("議事録")})

	var cliErr *Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("jqの適用の失敗を期待したが %v (%T) だった", err, err)
	}
	if cliErr.Code != CodeJQFailed || cliErr.ExitCode != ExitError {
		t.Errorf("コード = %q・終了コード = %d、期待値 = %q・%d", cliErr.Code, cliErr.ExitCode, CodeJQFailed, ExitError)
	}
	if want := "コマンド自体は完了しています。同じコマンドを再実行せず、読み取りのコマンドで結果を確かめてください。"; cliErr.Hint != want {
		t.Errorf("ヒント = %q、期待値 = %q", cliErr.Hint, want)
	}
	if got := stdout.String(); got != "" {
		t.Errorf("失敗したときは途中の結果も出さないことを期待したが、標準出力 = %q", got)
	}
}

func TestPrinterJQHaltError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		expr string
	}{
		{name: "nilの値を持つ非ゼロの停止も失敗になる", expr: "null | halt_error(5)"},
		{name: "既定の非ゼロの停止も失敗になる", expr: "null | halt_error"},
		{name: "停止の前に出した値も残さない", expr: ".number, (null | halt_error(5))"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p, stdout, err := newTestPrinter(t, printerSetup{args: []string{"--jq", tt.expr}, spec: pageSpec})
			if err != nil {
				t.Fatal(err)
			}

			err = p.PrintItem(testPage{Number: 12})

			var cliErr *Error
			if !errors.As(err, &cliErr) {
				t.Fatalf("jqの適用の失敗を期待したが %v (%T) だった", err, err)
			}
			if cliErr.Code != CodeJQFailed || ExitCodeOf(err) != ExitError {
				t.Errorf("コード = %q・終了コード = %d、期待値 = %q・%d", cliErr.Code, ExitCodeOf(err), CodeJQFailed, ExitError)
			}
			if want := "コマンド自体は完了しています。同じコマンドを再実行せず、読み取りのコマンドで結果を確かめてください。"; cliErr.Hint != want {
				t.Errorf("ヒント = %q、期待値 = %q", cliErr.Hint, want)
			}
			if got := stdout.String(); got != "" {
				t.Errorf("失敗したときは途中の結果も出さないことを期待したが、標準出力 = %q", got)
			}
		})
	}
}

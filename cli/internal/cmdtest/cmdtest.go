// Package cmdtestは、コマンドのテストに使う実行環境とAPIのスタブを提供する。
//
// コマンドはプロセスを起動せずに実行し、標準出力・標準エラー・返したエラーを確かめる。
// APIへの要求はURLを書き換えずにトランスポートで受け、メソッド・ホスト・パスで登録した応答を返す。
// 登録していない要求はテストを失敗させ、送るはずのない要求に気付けるようにする。
package cmdtest

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/config"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/iostreams"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// 環境変数で渡すトークン・スペース・接続先の既定値。
const (
	Token = "wikino_pat_test-secret"
	Space = "team"
	Host  = "https://wikino.example.com"
)

var update = flag.Bool("update", false, "ゴールデンファイルを今の出力で書き換える")

// Handlerは、登録した要求に応答を返す。
type Handler func(req *http.Request) *http.Response

// Envは、コマンドを実行する環境 (環境変数・APIのスタブ・入出力) を持つ。
type Env struct {
	t *testing.T

	// Varsはコマンドから見える環境変数。
	Vars map[string]string

	mu       sync.Mutex
	handlers map[string]Handler
	requests []*http.Request

	// Stdinはコマンドに渡す標準入力。
	Stdin string

	Stdout *bytes.Buffer
	Stderr *bytes.Buffer
}

// Newは、`WIKINO_TOKEN` と `WIKINO_SPACE` でスペースteamのAPIを呼ぶ環境を返す。表示言語は日本語にする。
func New(t *testing.T) *Env {
	t.Helper()

	return &Env{
		t: t,
		Vars: map[string]string{
			"XDG_CONFIG_HOME":    t.TempDir(),
			"WIKINO_LANG":        i18n.LangJa,
			config.EnvToken:      Token,
			config.EnvSpace:      Space,
			config.EnvHost:       Host,
			"WIKINO_TOKEN_STORE": "file",
		},
		handlers: map[string]Handler{},
	}
}

// Handleは、methodとpath (接続先からのパス。例: /api/v1/spaces/team/pages) の要求にhandlerで応答する。
func (e *Env) Handle(method, path string, handler Handler) {
	e.handlers[method+" "+path] = handler
}

// HandleURLは、APIの接続先でないURL (署名付きURLなど。例: https://storage.example.com/key) への
// methodの要求にhandlerで応答する。クエリは照合に含めない。この要求にAuthorizationヘッダーが付いていればテストを失敗させる。
func (e *Env) HandleURL(method, rawURL string, handler Handler) {
	e.handlers[method+" "+rawURL] = handler
}

// Requestsは、APIのスタブが受けた要求を返す。
func (e *Env) Requests() []*http.Request {
	e.mu.Lock()
	defer e.mu.Unlock()

	return append([]*http.Request(nil), e.requests...)
}

// Runは、標準入出力をTTYでない状態にしてargsでコマンドを実行する。
func (e *Env) Run(newCmd func(*cmdutil.Factory) *cobra.Command, args ...string) error {
	e.t.Helper()

	return e.execute(false, newCmd, args...)
}

// RunTTYは、端末から実行したときと同じく、標準入力と標準出力をTTYにしてargsでコマンドを実行する。
func (e *Env) RunTTY(newCmd func(*cmdutil.Factory) *cobra.Command, args ...string) error {
	e.t.Helper()

	return e.execute(true, newCmd, args...)
}

func (e *Env) execute(tty bool, newCmd func(*cmdutil.Factory) *cobra.Command, args ...string) error {
	e.t.Helper()

	ios, stdin, stdout, stderr := iostreams.Test()
	stdin.WriteString(e.Stdin)
	ios.SetStdinTTY(tty)
	ios.SetStdoutTTY(tty)
	// ゴールデンファイルを読みやすく保つため、人間向けの表示に色を付けない。
	ios.SetEnv([]string{"NO_COLOR=1"})
	e.Stdout, e.Stderr = stdout, stderr

	getenv := func(key string) string { return e.Vars[key] }
	f := &cmdutil.Factory{
		IOStreams:  ios,
		Localizer:  i18n.NewLocalizer(i18n.DetectLang(getenv)),
		Version:    "1.2.3",
		Getenv:     getenv,
		HTTPClient: &http.Client{Transport: roundTripFunc(e.serve)},
	}

	// ルートのコマンドは各コマンドのパッケージに依存するため使わず、同じ永続フラグを持つ親を組み立てる。
	root := &cobra.Command{Use: "wikino", SilenceErrors: true, SilenceUsage: true}
	output.AddFlags(root.PersistentFlags(), f.Localizer)
	config.AddFlags(root.PersistentFlags(), f.Localizer)
	root.AddCommand(newCmd(f))
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root.Execute()
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func (e *Env) serve(req *http.Request) (*http.Response, error) {
	e.mu.Lock()
	e.requests = append(e.requests, req)
	e.mu.Unlock()

	key := req.Method + " " + req.URL.Path
	if origin := req.URL.Scheme + "://" + req.URL.Host; origin != Host {
		// APIの接続先でないURL (署名付きURLなど) には、トークンを送らない。
		if got := req.Header.Get("Authorization"); got != "" {
			e.t.Errorf("%s への要求にAuthorizationヘッダー %q が付いている", origin, got)
		}
		key = req.Method + " " + origin + req.URL.Path
	} else if got, want := req.Header.Get("Authorization"), "Bearer "+Token; got != want {
		e.t.Errorf("Authorizationヘッダー = %q、期待値 = %q", got, want)
	}

	handler, ok := e.handlers[key]
	if !ok {
		e.t.Errorf("登録していない要求を受けた: %s %s", req.Method, req.URL)
		return Problem(http.StatusInternalServerError, "登録していない要求です。"), nil
	}

	return handler(req), nil
}

// JSONは、vを本文とするstatusの応答を返す。headerは応答のヘッダーに足す。
func JSON(status int, v any, header ...string) *http.Response {
	body, _ := json.Marshal(v)
	h := http.Header{"Content-Type": {"application/json"}}
	for i := 0; i+1 < len(header); i += 2 {
		h.Set(header[i], header[i+1])
	}

	return &http.Response{StatusCode: status, Header: h, Body: io.NopCloser(bytes.NewReader(body))}
}

// Problemは、detailを説明とするProblem Detailsの応答を返す。
func Problem(status int, detail string) *http.Response {
	body, _ := json.Marshal(map[string]any{"type": "about:blank", "title": http.StatusText(status), "status": status, "detail": detail})

	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/problem+json"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

// AssertCLIErrorは、errがwantCodeとwantExitCodeのCLIのエラーであることを確かめて返す。
func AssertCLIError(t *testing.T, err error, wantCode string, wantExitCode int) *output.Error {
	t.Helper()

	cliErr, ok := err.(*output.Error)
	if !ok {
		t.Fatalf("CLIのエラーを期待したが %v (%T) だった", err, err)
	}
	if cliErr.Code != wantCode || cliErr.ExitCode != wantExitCode {
		t.Errorf("コード・終了コード = %q, %d、期待値 = %q, %d (%s)", cliErr.Code, cliErr.ExitCode, wantCode, wantExitCode, cliErr.Message)
	}

	return cliErr
}

// AssertGoldenは、gotがtestdata/{name}.goldenと一致することを確かめる。
// `go test -update` で実行すると、gotでゴールデンファイルを書き換える。
func AssertGolden(t *testing.T, name, got string) {
	t.Helper()

	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}

	want, err := os.ReadFile(path) // #nosec G304 -- テストのゴールデンファイルの名前はテストが決める
	if err != nil {
		t.Fatalf("ゴールデンファイルを読めない (`go test -update` で作成できる): %v", err)
	}
	if got != string(want) {
		t.Errorf("出力がゴールデンファイル %s と違う\n--- 出力\n%s\n--- 期待値\n%s", path, got, want)
	}
}

// Decodeは、sをJSONとして読む。
func Decode(t *testing.T, s string, v any) {
	t.Helper()

	if err := json.Unmarshal([]byte(s), v); err != nil {
		t.Fatalf("JSONとして読めない: %v\n%s", err, s)
	}
}

// Linesは、sを改行で区切った行を返す。末尾の改行の後の空の行は含めない。
func Lines(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}

	return strings.Split(s, "\n")
}

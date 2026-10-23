package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/auth"
	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/config"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/iostreams"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

const (
	aliceToken = "wikino_pat_alice-secret"
	botToken   = "wikino_pat_bot-secret"
	carolToken = "wikino_pat_carol-secret"
)

// fakeKeyringは、メモリに保存するキーチェーン。errがあれば、すべての操作でそれを返す。
type fakeKeyring struct {
	items     map[string]string
	err       error
	getErr    error
	setErr    error
	deleteErr error
	beforeSet func()
}

func (k *fakeKeyring) Get(service, user string) (string, error) {
	if k.getErr != nil {
		return "", k.getErr
	}
	if k.err != nil {
		return "", k.err
	}
	v, ok := k.items[service+"/"+user]
	if !ok {
		return "", auth.ErrKeyringNotFound
	}
	return v, nil
}

func (k *fakeKeyring) Set(service, user, password string) error {
	if k.beforeSet != nil {
		k.beforeSet()
	}
	if k.setErr != nil {
		return k.setErr
	}
	if k.err != nil {
		return k.err
	}
	k.items[service+"/"+user] = password
	return nil
}

func (k *fakeKeyring) Delete(service, user string) error {
	if k.deleteErr != nil {
		return k.deleteErr
	}
	if k.err != nil {
		return k.err
	}
	delete(k.items, service+"/"+user)
	return nil
}

// tokenOwnerは、テスト用のAPIがトークンごとに返す持ち主とトークンの情報。
type tokenOwner struct {
	space    string
	memberID string
	atname   string
	name     string
}

var testOwners = map[string]tokenOwner{
	aliceToken: {space: "team", memberID: "11111111-1111-1111-1111-111111111111", atname: "alice", name: "Alice"},
	botToken:   {space: "team", memberID: "22222222-2222-2222-2222-222222222222", atname: "bot", name: ""},
	carolToken: {space: "solo", memberID: "33333333-3333-3333-3333-333333333333", atname: "carol", name: "Carol"},
}

// roundTripFuncは関数をトランスポートにする。URLを書き換えずに要求を検証し、応答を返すために使う。
type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// testEnvは、コマンドを実行する環境 (設定のディレクトリ・キーチェーン・API・入出力) を持つ。
type testEnv struct {
	t              *testing.T
	vars           map[string]string
	keyring        *fakeKeyring
	owners         map[string]tokenOwner
	requests       []*http.Request
	beforeResponse func()
	oauth          *fakeOAuth
	// browserは、ブラウザで認可要求のURLを開く代わりに呼ぶ。nilならブラウザで許可する。
	browser func(authorizationURL string) error
	now     time.Time
	// waitsは、デバイス認可のポーリングで待った時間。実際には待たない。
	waits []time.Duration

	ios    *iostreams.IOStreams
	stdin  *bytes.Buffer
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	e := &testEnv{
		t: t,
		// ディスプレイが無いLinuxではデバイス認可に切り替わるため、ディスプレイがある環境にする。
		vars:    map[string]string{"XDG_CONFIG_HOME": t.TempDir(), "WIKINO_LANG": i18n.LangJa, "DISPLAY": ":0"},
		keyring: &fakeKeyring{items: map[string]string{}},
		owners:  map[string]tokenOwner{},
		oauth:   newFakeOAuth(),
		now:     time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC),
	}
	for k, v := range testOwners {
		e.owners[k] = v
	}

	return e
}

// runはargsでwikinoを実行する。stdinは標準入力に渡す内容。標準入出力はTTYでない。
func (e *testEnv) run(stdin string, args ...string) error {
	e.t.Helper()

	return e.execute(false, stdin, args...)
}

// runTTYは、標準出力をTTYにしてargsでwikinoを実行する。
func (e *testEnv) runTTY(stdin string, args ...string) error {
	e.t.Helper()

	return e.execute(true, stdin, args...)
}

func (e *testEnv) execute(tty bool, stdin string, args ...string) error {
	e.t.Helper()

	e.ios, e.stdin, e.stdout, e.stderr = iostreams.Test()
	e.ios.SetStdoutTTY(tty)
	e.stdin.WriteString(stdin)

	f := e.factory()
	loc := f.Localizer

	// ルートのコマンドはこのパッケージに依存するため使えない。同じ永続フラグを持つ親を組み立てる。
	root := &cobra.Command{Use: "wikino", SilenceErrors: true, SilenceUsage: true}
	output.AddFlags(root.PersistentFlags(), loc)
	config.AddFlags(root.PersistentFlags(), loc)
	root.AddCommand(NewCmdAuth(f))
	root.SetArgs(args)
	root.SetOut(e.stdout)
	root.SetErr(e.stderr)

	return root.Execute()
}

// factoryは、保存処理をコマンドと同じ差し替え可能な環境で検証するために使う。
func (e *testEnv) factory() *cmdutil.Factory {
	f := &cmdutil.Factory{
		IOStreams:  e.ios,
		Localizer:  i18n.NewLocalizer(i18n.DetectLang(func(key string) string { return e.vars[key] })),
		Version:    "1.2.3",
		Getenv:     func(key string) string { return e.vars[key] },
		HTTPClient: &http.Client{Transport: roundTripFunc(e.serve)},
		OpenBrowser: func(authorizationURL string) error {
			if e.browser != nil {
				return e.browser(authorizationURL)
			}
			e.approve(authorizationURL, nil)
			return nil
		},
		Now: func() time.Time { return e.now },
		Sleep: func(_ context.Context, d time.Duration) error {
			e.waits = append(e.waits, d)
			return nil
		},
	}
	if e.keyring != nil {
		f.Keyring = e.keyring
	}
	return f
}

// serveはテスト用のAPI。トークンの持ち主のメンバーとトークンの情報を返す。
func (e *testEnv) serve(req *http.Request) (*http.Response, error) {
	e.requests = append(e.requests, req)
	if e.beforeResponse != nil {
		e.beforeResponse()
	}
	if strings.HasPrefix(req.URL.Path, "/.well-known/") || strings.HasPrefix(req.URL.Path, "/oauth/") {
		return e.oauth.serve(req, e.owners), nil
	}

	token := strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
	owner, ok := e.owners[token]
	if !ok {
		return problem(http.StatusUnauthorized, "トークンが無効です。"), nil
	}

	path := strings.TrimPrefix(req.URL.Path, "/api/v1/spaces/")
	space, rest, _ := strings.Cut(path, "/")
	if !strings.EqualFold(space, owner.space) {
		return problem(http.StatusNotFound, "スペースが見つかりません。"), nil
	}

	switch rest {
	case "members/me":
		return jsonResponse(map[string]any{"id": owner.memberID, "atname": owner.atname, "name": owner.name}), nil
	case "tokens/current":
		return jsonResponse(map[string]any{
			"kind":       "personal_access_token",
			"scopes":     []string{"page:read", "draft_page:read"},
			"expires_at": "2026-12-01T00:00:00Z",
		}), nil
	default:
		return problem(http.StatusNotFound, "見つかりません。"), nil
	}
}

func jsonResponse(v any) *http.Response {
	body, _ := json.Marshal(v)

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

func problem(status int, detail string) *http.Response {
	body, _ := json.Marshal(map[string]any{"type": "about:blank", "title": http.StatusText(status), "status": status, "detail": detail})

	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/problem+json"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

// loginはtokenでspaceにログインする。
func (e *testEnv) login(space, token string, args ...string) {
	e.t.Helper()

	if err := e.run(token+"\n", append([]string{"auth", "login", space, "--with-token"}, args...)...); err != nil {
		e.t.Fatalf("ログインに失敗した: %v", err)
	}
}

func (e *testEnv) config() *config.Config {
	e.t.Helper()

	cfg, err := config.Load(e.vars["XDG_CONFIG_HOME"] + "/wikino")
	if err != nil {
		e.t.Fatal(err)
	}

	return cfg
}

// decodeは標準出力のJSONを読む。
func (e *testEnv) decode(v any) {
	e.t.Helper()

	if err := json.Unmarshal(e.stdout.Bytes(), v); err != nil {
		e.t.Fatalf("標準出力をJSONとして読めない: %v\n%s", err, e.stdout.String())
	}
}

// assertNoSecretは、標準出力と標準エラーにトークンが含まれないことを確かめる。
func (e *testEnv) assertNoSecret() {
	e.t.Helper()

	secrets := []string{aliceToken, botToken, carolToken}
	secrets = append(secrets, e.oauth.secrets()...)
	for _, token := range secrets {
		if strings.Contains(e.stdout.String(), token) || strings.Contains(e.stderr.String(), token) {
			e.t.Errorf("出力にトークンが含まれる:\n標準出力: %s\n標準エラー: %s", e.stdout.String(), e.stderr.String())
		}
	}
}

func assertCLIError(t *testing.T, err error, wantCode string, wantExitCode int) *output.Error {
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

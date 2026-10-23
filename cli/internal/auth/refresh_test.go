package auth

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/cli/internal/config"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

var testNow = time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)

// newOAuthStoreは、serverを認可サーバーとし、ファイルに保存するStoreを返す。
func newOAuthStore(t *testing.T, dir string, server *fakeAuthServer, now time.Time) *Store {
	t.Helper()

	store, _ := newTestStore(t, nil, nil)
	store.Dir = dir
	store.OAuth = func(host string) *OAuthClient {
		return &OAuthClient{Host: host, HTTPClient: server.client()}
	}
	store.Now = func() time.Time { return now }

	return store
}

// loginOAuthは、期限がexpiresAtのOAuthのトークンを保存し、設定に資格情報を足す。
func loginOAuth(t *testing.T, store *Store, server *fakeAuthServer, expiresAt time.Time) *config.Credential {
	t.Helper()

	cred := testCredential()
	server.addRefreshToken("wikino_oauth_refresh_0")
	secret := Secret{AccessToken: "wikino_oauth_access_0", RefreshToken: "wikino_oauth_refresh_0", ExpiresAt: expiresAt}
	var err error
	if cred.Store, err = store.Save(cred, secret); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	cfg.AddCredential(*cred)
	if err := cfg.Save(store.Dir); err != nil {
		t.Fatal(err)
	}

	return cred
}

func tokenOf(t *testing.T, store *Store, cred *config.Credential) (string, error) {
	t.Helper()

	source, err := store.TokenSource(&config.Target{Host: cred.Host, Space: cred.Space, Credential: cred})
	if err != nil {
		t.Fatal(err)
	}

	return source.Token(context.Background())
}

func TestTokenSourceUsesFreshToken(t *testing.T) {
	t.Parallel()

	server := newFakeAuthServer()
	store := newOAuthStore(t, t.TempDir(), server, testNow)
	cred := loginOAuth(t, store, server, testNow.Add(10*time.Minute))

	token, err := tokenOf(t, store, cred)
	if err != nil || token != "wikino_oauth_access_0" {
		t.Fatalf("トークン = %q, エラー = %v", token, err)
	}
	if len(server.requests) != 0 {
		t.Error("期限まで間があるのに認可サーバーに要求した")
	}
}

func TestTokenSourceRefreshesBeforeExpiry(t *testing.T) {
	t.Parallel()

	server := newFakeAuthServer()
	store := newOAuthStore(t, t.TempDir(), server, testNow)
	// 期限の30秒前は、要求の途中で切れないよう更新する。
	cred := loginOAuth(t, store, server, testNow.Add(30*time.Second))

	token, err := tokenOf(t, store, cred)
	if err != nil || token != "wikino_oauth_access_1" {
		t.Fatalf("トークン = %q, エラー = %v", token, err)
	}
	form := server.forms[0]
	if form.Get("refresh_token") != "wikino_oauth_refresh_0" || form.Get("resource") != "https://wikino.app/api/v1/spaces/Team" {
		t.Errorf("リフレッシュの要求 = %v", form)
	}

	// 更新したトークンは、世代を変えずに同じ保存先へ書き、次の実行で使う。
	saved, err := store.Load(cred)
	if err != nil {
		t.Fatal(err)
	}
	if saved.AccessToken != "wikino_oauth_access_1" || saved.RefreshToken != "wikino_oauth_refresh_1" ||
		!saved.ExpiresAt.Equal(testNow.Add(time.Hour)) || strings.Join(saved.Scopes, " ") != "page:read draft_page:read" {
		t.Errorf("保存したトークン = %+v", saved)
	}
}

func TestTokenSourceParallelRefreshUsesRefreshTokenOnce(t *testing.T) {
	t.Parallel()

	server := newFakeAuthServer()
	dir := t.TempDir()
	cred := loginOAuth(t, newOAuthStore(t, dir, server, testNow), server, testNow.Add(-time.Minute))

	// 並列に起動したCLIを、別々のStore (別々のロックファイルの記述子) で表す。
	const workers = 8
	tokens := make([]string, workers)
	errs := make([]error, workers)
	var wg sync.WaitGroup
	for i := range workers {
		wg.Go(func() {
			store := newOAuthStore(t, dir, server, testNow)
			source, err := store.TokenSource(&config.Target{Host: cred.Host, Space: cred.Space, Credential: cred})
			if err != nil {
				errs[i] = err
				return
			}
			tokens[i], errs[i] = source.Token(context.Background())
		})
	}
	wg.Wait()

	for i := range workers {
		if errs[i] != nil || tokens[i] != "wikino_oauth_access_1" {
			t.Errorf("%d番目: トークン = %q, エラー = %v", i, tokens[i], errs[i])
		}
	}
	if server.refreshCount != 1 || server.usedRefreshCount != 0 {
		t.Errorf("リフレッシュの数 = %d、使用済みの再提示 = %d。期待値 = 1, 0", server.refreshCount, server.usedRefreshCount)
	}
}

// setFailingKeyringは、ログインの後に書き込みだけが失敗するキーチェーン。
type setFailingKeyring struct {
	*fakeKeyring
	setErr error
}

func (k *setFailingKeyring) Set(service, user, password string) error {
	if k.setErr != nil {
		return k.setErr
	}
	return k.fakeKeyring.Set(service, user, password)
}

func TestTokenSourceRefreshSaveFailureUsesNewToken(t *testing.T) {
	t.Parallel()

	server := newFakeAuthServer()
	store := newOAuthStore(t, t.TempDir(), server, testNow)
	kr := &setFailingKeyring{fakeKeyring: newFakeKeyring()}
	store.Keyring = kr
	var warn bytes.Buffer
	store.Warn = &warn
	cred := loginOAuth(t, store, server, testNow.Add(-time.Minute))
	kr.setErr = errors.New("キーチェーンに書き込めません")

	// 古いリフレッシュトークンは使用済みのため、保存できなくても得たトークンで今回の要求を続ける。
	token, err := tokenOf(t, store, cred)
	if err != nil || token != "wikino_oauth_access_1" {
		t.Fatalf("トークン = %q, エラー = %v", token, err)
	}
	if !strings.Contains(warn.String(), "wikino auth login Team") {
		t.Errorf("ログインし直しの警告を期待したが %q だった", warn.String())
	}
	if strings.Contains(warn.String(), "wikino_oauth_") {
		t.Error("警告にトークンが含まれる")
	}
}

func TestTokenSourceRefreshRejected(t *testing.T) {
	t.Parallel()

	server := newFakeAuthServer()
	store := newOAuthStore(t, t.TempDir(), server, testNow)
	cred := loginOAuth(t, store, server, testNow.Add(-time.Minute))
	// 連携の解除などで、保存したリフレッシュトークンが使えなくなった状態にする。
	server.refreshTokens = map[string]bool{}

	_, err := tokenOf(t, store, cred)
	cliErr, ok := err.(*output.Error)
	if !ok || cliErr.Code != output.CodeUnauthorized || cliErr.ExitCode != output.ExitAuth {
		t.Fatalf("エラー = %v", err)
	}
	if !strings.Contains(cliErr.Hint, "wikino auth login Team") {
		t.Errorf("ヒント = %q", cliErr.Hint)
	}
	if strings.Contains(cliErr.Message+cliErr.Hint, "wikino_oauth_") {
		t.Error("エラーにトークンが含まれる")
	}
}

func TestTokenSourceRefreshAfterLogout(t *testing.T) {
	t.Parallel()

	server := newFakeAuthServer()
	store := newOAuthStore(t, t.TempDir(), server, testNow)
	cred := loginOAuth(t, store, server, testNow.Add(-time.Minute))
	source, err := store.TokenSource(&config.Target{Host: cred.Host, Space: cred.Space, Credential: cred})
	if err != nil {
		t.Fatal(err)
	}

	// トークンを読んだ後、更新する前に別のプロセスがログアウトした。
	if err := (&config.Config{}).Save(store.Dir); err != nil {
		t.Fatal(err)
	}

	_, err = source.Token(context.Background())
	assertNotLoggedIn(t, err)
	if len(server.forms) != 0 {
		t.Error("ログアウトした資格情報のリフレッシュトークンを使った")
	}
}

func TestTokenSourceRefreshRequestTimeout(t *testing.T) {
	t.Parallel()

	server := newFakeAuthServer()
	store := newOAuthStore(t, t.TempDir(), server, testNow)
	cred := loginOAuth(t, store, server, testNow.Add(-time.Minute))
	// ロックを取った後の要求が、HTTPクライアントの時間の上限を超えた。
	store.OAuth = func(host string) *OAuthClient {
		client := &http.Client{
			Timeout: 20 * time.Millisecond,
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				<-req.Context().Done()
				return nil, req.Context().Err()
			}),
		}
		return &OAuthClient{Host: host, HTTPClient: client}
	}

	// 時間切れのエラーもcontext.DeadlineExceededを包むが、ロック待ちの中止ではなく通信の失敗にする。
	_, err := tokenOf(t, store, cred)
	cliErr, ok := err.(*output.Error)
	if !ok || cliErr.Code != output.CodeNetwork || cliErr.ExitCode != output.ExitError {
		t.Fatalf("エラー = %v、期待するコード = %s", err, output.CodeNetwork)
	}
}

func TestTokenSourceCanceledWhileWaitingForLock(t *testing.T) {
	t.Parallel()

	server := newFakeAuthServer()
	store := newOAuthStore(t, t.TempDir(), server, testNow)
	cred := loginOAuth(t, store, server, testNow.Add(-time.Minute))
	source, err := store.TokenSource(&config.Target{Host: cred.Host, Space: cred.Space, Credential: cred})
	if err != nil {
		t.Fatal(err)
	}

	// 別のプロセスが同じ資格情報のロックを持っている間に、待つのを打ち切った。
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	locked := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- WithCredentialLock(context.Background(), store.Dir, cred, func() error {
			close(locked)
			<-release
			return nil
		})
	}()
	<-locked
	_, err = source.Token(ctx)
	close(release)
	if lockErr := <-done; lockErr != nil {
		t.Fatal(lockErr)
	}

	cliErr, ok := err.(*output.Error)
	if !ok || cliErr.Code != output.CodeUnexpected || cliErr.Message != "トークンの更新を待つ間に中止しました。" {
		t.Fatalf("エラー = %v、ロック待ちの中止を期待した", err)
	}
	if len(server.forms) != 0 {
		t.Error("ロックを取れないままリフレッシュした")
	}
}

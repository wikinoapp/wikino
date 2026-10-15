package auth

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/cli/internal/config"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

const testToken = "wikino_pat_secret-token-value"

// fakeKeyringは、メモリに保存するKeyring。errがあれば、すべての操作でそれを返す。
type fakeKeyring struct {
	items map[string]string
	err   error
}

func newFakeKeyring() *fakeKeyring {
	return &fakeKeyring{items: map[string]string{}}
}

func (k *fakeKeyring) Get(service, user string) (string, error) {
	if k.err != nil {
		return "", k.err
	}
	v, ok := k.items[service+"/"+user]
	if !ok {
		return "", ErrKeyringNotFound
	}
	return v, nil
}

func (k *fakeKeyring) Set(service, user, password string) error {
	if k.err != nil {
		return k.err
	}
	k.items[service+"/"+user] = password
	return nil
}

func (k *fakeKeyring) Delete(service, user string) error {
	if k.err != nil {
		return k.err
	}
	if _, ok := k.items[service+"/"+user]; !ok {
		return ErrKeyringNotFound
	}
	delete(k.items, service+"/"+user)
	return nil
}

func testCredential() *config.Credential {
	return &config.Credential{Host: config.DefaultHost, Space: "Team", MemberID: "id-alice", MemberAtname: "alice"}
}

func newTestStore(t *testing.T, kr Keyring, env map[string]string) (*Store, *bytes.Buffer) {
	t.Helper()

	var warn bytes.Buffer

	return &Store{
		Keyring: kr,
		Dir:     filepath.Join(t.TempDir(), "wikino"),
		Getenv:  func(key string) string { return env[key] },
		Warn:    &warn,
		Loc:     i18n.NewLocalizer(i18n.LangJa),
	}, &warn
}

func TestStoreKeyring(t *testing.T) {
	t.Parallel()

	kr := newFakeKeyring()
	store, warn := newTestStore(t, kr, nil)
	cred := testCredential()

	got, err := store.Save(cred, Secret{AccessToken: testToken})
	if err != nil || got != StoreKeyring {
		t.Fatalf("保存先 = %q, %v、期待値 = keyring", got, err)
	}
	if warn.Len() != 0 {
		t.Errorf("警告を期待しなかったが %q だった", warn.String())
	}
	if _, err := os.Stat(store.Dir); !errors.Is(err, os.ErrNotExist) {
		t.Error("キーチェーンに保存したのにファイルを作った")
	}

	cred.Store = got
	// スペースの識別子の大文字と小文字が違っても同じ項目を指す。
	other := *cred
	other.Space = "team"
	secret, err := store.Load(&other)
	if err != nil || secret.AccessToken != testToken {
		t.Fatalf("取り出したトークン = %q, %v", secret.AccessToken, err)
	}

	if err := store.Delete(cred); err != nil {
		t.Fatalf("削除に失敗した: %v", err)
	}
	if len(kr.items) != 0 {
		t.Errorf("キーチェーンに項目が残った: %v", kr.items)
	}
	if err := store.Delete(cred); err != nil {
		t.Errorf("既に無い項目の削除でエラーを期待しなかったが %v だった", err)
	}
	assertNotLoggedIn(t, func() error { _, err := store.Load(cred); return err }())
}

func TestStoreFallsBackToFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		keyring  Keyring
		env      map[string]string
		wantWarn bool
	}{
		{name: "キーチェーンが使えなければ警告してファイルに保存する", keyring: &fakeKeyring{err: errors.New("dbus: no session bus")}, wantWarn: true},
		{name: "キーチェーンが無ければファイルに保存する", keyring: nil},
		{name: "WIKINO_TOKEN_STOREがfileならキーチェーンを使わない", keyring: newFakeKeyring(), env: map[string]string{EnvTokenStore: "file"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store, warn := newTestStore(t, tt.keyring, tt.env)
			cred := testCredential()

			got, err := store.Save(cred, Secret{AccessToken: testToken})
			if err != nil || got != StoreFile {
				t.Fatalf("保存先 = %q, %v、期待値 = file", got, err)
			}
			if kr, ok := tt.keyring.(*fakeKeyring); ok && len(kr.items) != 0 {
				t.Error("キーチェーンに保存した")
			}
			if gotWarn := strings.Contains(warn.String(), "キーチェーンを使えない"); gotWarn != tt.wantWarn {
				t.Errorf("警告 = %q、警告を期待するか = %v", warn.String(), tt.wantWarn)
			}
			if strings.Contains(warn.String(), testToken) {
				t.Error("警告にトークンが含まれる")
			}

			path := store.filePath(cred)
			if runtime.GOOS != "windows" {
				info, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				if info.Mode().Perm() != 0o600 {
					t.Errorf("トークンのファイルのパーミッション = %o、期待値 = 600", info.Mode().Perm())
				}
			}

			cred.Store = got
			secret, err := store.Load(cred)
			if err != nil || secret.AccessToken != testToken {
				t.Fatalf("取り出したトークン = %q, %v", secret.AccessToken, err)
			}

			if err := store.Delete(cred); err != nil {
				t.Fatalf("削除に失敗した: %v", err)
			}
			if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
				t.Error("トークンのファイルが残った")
			}
		})
	}
}

func TestStoreFallbackFailureDoesNotWarnSaved(t *testing.T) {
	t.Parallel()

	store, warn := newTestStore(t, &fakeKeyring{err: errors.New("dbus: no session bus")}, nil)
	// トークンを置くディレクトリの場所をファイルにして、ファイルへの保存も失敗させる。
	if err := os.MkdirAll(store.Dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.credentialsDir(), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Save(testCredential(), Secret{AccessToken: testToken}); err == nil {
		t.Fatal("ファイルへの保存の失敗を期待")
	}
	if strings.Contains(warn.String(), "キーチェーンを使えない") {
		t.Errorf("保存できなかったのにファイルへ保存したと警告した: %q", warn.String())
	}
}

func TestStoreWarnsLoosePermissions(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("Windowsはパーミッションのビットでアクセスを制御しない")
	}

	store, warn := newTestStore(t, nil, nil)
	cred := testCredential()
	if _, err := store.Save(cred, Secret{AccessToken: testToken}); err != nil {
		t.Fatal(err)
	}
	cred.Store = StoreFile

	if _, err := store.Load(cred); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(warn.String(), "本人以外") {
		t.Fatalf("パーミッションが0700・0600なのに警告した: %q", warn.String())
	}

	if err := os.Chmod(store.Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(store.filePath(cred), 0o644); err != nil {
		t.Fatal(err)
	}
	warn.Reset()

	if _, err := store.Load(cred); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"chmod 700 " + store.Dir, "chmod 600 " + store.filePath(cred)} {
		if !strings.Contains(warn.String(), want) {
			t.Errorf("警告に %q を含むことを期待したが %q だった", want, warn.String())
		}
	}
}

func TestStoreWarnsLooseCredentialsDirectory(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("WindowsはACLでアクセスを制御する")
	}
	store, warn := newTestStore(t, nil, nil)
	cred := testCredential()
	var err error
	cred.Store, err = store.Save(cred, Secret{AccessToken: testToken})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(store.credentialsDir(), 0o777); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(cred); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(warn.String(), "chmod 700 "+store.credentialsDir()) {
		t.Fatalf("資格情報ディレクトリの警告がない: %q", warn.String())
	}
}

func TestStoreInvalidTokenStore(t *testing.T) {
	t.Parallel()

	store, _ := newTestStore(t, newFakeKeyring(), map[string]string{EnvTokenStore: "plaintext"})

	_, err := store.Save(testCredential(), Secret{AccessToken: testToken})

	var cliErr *output.Error
	if !errors.As(err, &cliErr) || cliErr.ExitCode != output.ExitUsage {
		t.Fatalf("使い方の誤りを期待したが %v だった", err)
	}
}

func TestStoreLoadKeyringError(t *testing.T) {
	t.Parallel()

	store, _ := newTestStore(t, &fakeKeyring{err: errors.New("keyring is locked")}, nil)
	store.Loc = i18n.NewLocalizer(i18n.LangEn)
	cred := testCredential()
	cred.Store = StoreKeyring

	_, err := store.Load(cred)

	var cliErr *output.Error
	if !errors.As(err, &cliErr) || cliErr.Code != output.CodeUnexpected || cliErr.ExitCode != output.ExitError {
		t.Fatalf("キーチェーンの読み取りの失敗を期待したが %v だった", err)
	}
	if cliErr.Message != "Could not read the token from the keychain." || cliErr.Hint == "" {
		t.Errorf("表示言語に翻訳されていない: %q / %q", cliErr.Message, cliErr.Hint)
	}
}

func TestStoreLoadMissingFile(t *testing.T) {
	t.Parallel()

	store, _ := newTestStore(t, nil, nil)
	cred := testCredential()
	cred.Store = StoreFile

	_, err := store.Load(cred)
	assertNotLoggedIn(t, err)
}

func TestTokenSource(t *testing.T) {
	t.Parallel()

	store, _ := newTestStore(t, newFakeKeyring(), nil)
	cred := testCredential()
	var err error
	if cred.Store, err = store.Save(cred, Secret{AccessToken: testToken}); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		target *config.Target
		want   string
	}{
		{name: "WIKINO_TOKENがあればそれを使う", target: &config.Target{EnvToken: "env-token", Credential: cred}, want: "env-token"},
		{name: "無ければ保存したトークンを使う", target: &config.Target{Credential: cred}, want: testToken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			source, err := store.TokenSource(tt.target)
			if err != nil {
				t.Fatal(err)
			}
			got, err := source.Token(context.Background())
			if err != nil || got != tt.want {
				t.Errorf("トークン = %q, %v、期待値 = %q", got, err, tt.want)
			}
		})
	}
}

func assertNotLoggedIn(t *testing.T, err error) {
	t.Helper()

	var cliErr *output.Error
	if !errors.As(err, &cliErr) || cliErr.Code != output.CodeNotLoggedIn || cliErr.ExitCode != output.ExitAuth {
		t.Fatalf("未ログインのエラーを期待したが %v だった", err)
	}
}

func TestStoreTimedOutSaveRetainsCleanupTarget(t *testing.T) {
	t.Parallel()
	backend := &delayedKeyring{started: make(chan struct{}), release: make(chan struct{}), finished: make(chan struct{})}
	k := &systemKeyring{backend: backend, timeout: 30 * time.Millisecond, getenv: keyringTestEnv(t.TempDir())}
	store, _ := newTestStore(t, k, nil)
	cred := testCredential()
	var err error
	cred.Store, err = store.Save(cred, Secret{AccessToken: testToken})
	if err != nil {
		t.Fatal(err)
	}
	if cred.Store != StoreFile || cred.CleanupStore != StoreKeyring {
		t.Fatalf("時間切れ後の保存先 = %+v", cred)
	}
	if err := store.Cleanup(cred); !errors.Is(err, errKeyringTimeout) {
		t.Fatalf("未完了の保存中の削除 = %v、時間切れを期待", err)
	}
	if cred.CleanupStore != StoreKeyring {
		t.Fatal("削除失敗で再試行先が消えた")
	}
	close(backend.release)
	<-backend.finished
	if err := store.Cleanup(cred); err != nil {
		t.Fatal(err)
	}
	if cred.CleanupStore != "" {
		t.Fatal("削除成功後も再試行先が残る")
	}
	secret, err := store.Load(cred)
	if err != nil || secret.AccessToken != testToken {
		t.Fatalf("新しい保存先のトークンが消えた: %v", err)
	}
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.value != "" {
		t.Fatal("遅れて保存したキーチェーントークンが残る")
	}
}

func TestStoreLateSetKeepsLatestLogin(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "keyring-token")
	backend := &blockedSetKeyring{Keyring: fileKeyring{path: path}, started: make(chan struct{}), release: make(chan struct{}), finished: make(chan struct{})}
	var release sync.Once
	t.Cleanup(func() { release.Do(func() { close(backend.release) }) })
	first, _ := newTestStore(t, &systemKeyring{backend: backend, timeout: 100 * time.Millisecond, getenv: keyringTestEnv(dir)}, nil)
	second, _ := newTestStore(t, &systemKeyring{backend: fileKeyring{path: path}, timeout: 100 * time.Millisecond, getenv: keyringTestEnv(dir)}, nil)
	second.Dir = first.Dir
	login := func(store *Store, token string) {
		t.Helper()
		if err := config.WithLocked(context.Background(), store.Dir, func(cfg *config.Config) error {
			cred := testCredential()
			var err error
			cred.Store, err = store.Save(cred, Secret{AccessToken: token})
			if err != nil {
				return err
			}
			cfg.AddCredential(*cred)
			if err := cfg.Save(store.Dir); err != nil {
				return err
			}
			if err := store.Cleanup(&cfg.Credentials[0]); !errors.Is(err, errKeyringTimeout) {
				t.Fatalf("未完了の保存中の削除 = %v、時間切れを期待", err)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	// 設定ロックを解放した後に、別のラッパーから再ログインする。
	login(first, "first-token")
	<-backend.started
	login(second, "second-token")
	cfg, err := config.Load(second.Dir)
	if err != nil {
		t.Fatal(err)
	}
	cred := &cfg.Credentials[0]
	if cred.Store != StoreFile || cred.CleanupStore != StoreKeyring {
		t.Fatalf("再ログインの保存先 = %+v", cred)
	}
	release.Do(func() { close(backend.release) })
	<-backend.finished
	if err := second.Cleanup(cred); err != nil {
		t.Fatal(err)
	}
	secret, err := second.Load(cred)
	if err != nil || secret.AccessToken != "second-token" {
		t.Fatalf("最新のトークン = %q, %v", secret.AccessToken, err)
	}
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("遅れて保存した旧トークンが残った: %v", err)
	}
}

func TestStoreLogoutWaitsForLateSet(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "keyring-token")
	backend := &blockedSetKeyring{Keyring: fileKeyring{path: path}, started: make(chan struct{}), release: make(chan struct{}), finished: make(chan struct{})}
	var release sync.Once
	t.Cleanup(func() { release.Do(func() { close(backend.release) }) })
	first, _ := newTestStore(t, &systemKeyring{backend: backend, timeout: 100 * time.Millisecond, getenv: keyringTestEnv(dir)}, nil)
	second, _ := newTestStore(t, &systemKeyring{backend: fileKeyring{path: path}, timeout: 100 * time.Millisecond, getenv: keyringTestEnv(dir)}, nil)
	second.Dir = first.Dir
	cred := testCredential()
	var err error
	cred.Store, err = first.Save(cred, Secret{AccessToken: testToken})
	if err != nil {
		t.Fatal(err)
	}
	<-backend.started
	// ログアウトは先行する保存が未完了なら失敗し、設定に残る削除先を再試行できる。
	if err := second.Delete(cred); !errors.Is(err, errKeyringTimeout) {
		t.Fatalf("未完了の保存中のログアウト = %v、時間切れを期待", err)
	}
	release.Do(func() { close(backend.release) })
	<-backend.finished
	if err := second.Delete(cred); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("ログアウト後にキーチェーンのトークンが残った: %v", err)
	}
	if _, err := os.Stat(second.filePath(cred)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("ログアウト後にファイルのトークンが残った: %v", err)
	}
}

func TestStoreLoadRejectsTokenFromOtherGeneration(t *testing.T) {
	t.Parallel()
	kr := newFakeKeyring()
	store, _ := newTestStore(t, kr, nil)
	first := testCredential()
	var err error
	if first.Store, err = store.Save(first, Secret{AccessToken: "first-token"}); err != nil {
		t.Fatal(err)
	}
	stale := kr.items[keyringService+"/"+key(first)]
	second := testCredential()
	if second.Store, err = store.Save(second, Secret{AccessToken: testToken}); err != nil {
		t.Fatal(err)
	}
	if second.Generation == "" || second.Generation == first.Generation {
		t.Fatalf("保存ごとに新しい世代を期待: %q, %q", first.Generation, second.Generation)
	}
	if secret, err := store.Load(second); err != nil || secret.AccessToken != testToken {
		t.Fatalf("取り出したトークン = %q, %v", secret.AccessToken, err)
	}
	// 前のログインの時間切れのSetが、後から同じ項目を書き戻した状態。
	kr.items[keyringService+"/"+key(second)] = stale
	assertNotLoggedIn(t, func() error { _, err := store.Load(second); return err }())
}

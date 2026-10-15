package auth

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/cli/internal/config"
)

// fileKeyringは、別プロセスから同じ保存先を使うためのテスト用キーチェーン。
type fileKeyring struct{ path string }

func (k fileKeyring) Get(_, _ string) (string, error) {
	data, err := os.ReadFile(k.path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", ErrKeyringNotFound
	}
	return string(data), err
}
func (k fileKeyring) Set(_, _, value string) error { return os.WriteFile(k.path, []byte(value), 0600) }
func (k fileKeyring) Delete(_, _ string) error {
	err := os.Remove(k.path)
	if errors.Is(err, fs.ErrNotExist) {
		return ErrKeyringNotFound
	}
	return err
}

// blockedSetKeyringは、OS操作が時間切れ後に完了する順序を固定する。
type blockedSetKeyring struct {
	Keyring
	started  chan struct{}
	release  chan struct{}
	finished chan struct{}
}

func (k *blockedSetKeyring) Set(service, user, value string) error {
	close(k.started)
	<-k.release
	defer close(k.finished)
	return k.Keyring.Set(service, user, value)
}

func TestSystemKeyringLateSetSerializesProcesses(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "token")
	backend := &blockedSetKeyring{Keyring: fileKeyring{path: path}, started: make(chan struct{}), release: make(chan struct{}), finished: make(chan struct{})}
	var release sync.Once
	t.Cleanup(func() { release.Do(func() { close(backend.release) }) })
	k := &systemKeyring{backend: backend, timeout: 100 * time.Millisecond, getenv: keyringTestEnv(dir)}
	if err := k.Set("service", "user", "first-token"); !errors.Is(err, errKeyringTimeout) {
		t.Fatalf("保存の時間切れを期待: %v", err)
	}
	<-backend.started
	run := func(action, expectation string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestSystemKeyringProcess$")
		cmd.Env = append(os.Environ(), "WIKINO_TEST_KEYRING_DIR="+dir, "WIKINO_TEST_KEYRING_ACTION="+action, "WIKINO_TEST_KEYRING_EXPECT="+expectation)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("子プロセス (%s) = %v\n%s", action, err, output)
		}
	}
	// 呼び出し側へ時間切れを返しても、別プロセスの保存・削除・読み取りは先行操作を追い越さない。
	for _, action := range []string{"set", "delete", "get"} {
		run(action, "timeout")
	}
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("未完了のSetを子プロセスが追い越した: %v", err)
	}
	release.Do(func() { close(backend.release) })
	<-backend.finished
	// 期限切れの待機操作は後から実行せず、先行Set完了後の再試行だけを反映する。
	run("set", "success")
	value, err := (fileKeyring{path: path}).Get("service", "user")
	if err != nil || value != "second-token" {
		t.Fatalf("再ログイン後のトークン = %q, %v", value, err)
	}
	run("delete", "success")
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("ログアウト後にトークンが残った: %v", err)
	}
}

// TestSystemKeyringProcessは、独立したラッパーが同じOSロックを共有することを検証する。
func TestSystemKeyringProcess(t *testing.T) {
	t.Parallel()
	dir := os.Getenv("WIKINO_TEST_KEYRING_DIR")
	if dir == "" {
		t.Skip("親テストから子プロセスとして実行する")
	}
	k := &systemKeyring{backend: fileKeyring{path: filepath.Join(dir, "token")}, timeout: 100 * time.Millisecond, getenv: keyringTestEnv(dir)}
	var err error
	switch os.Getenv("WIKINO_TEST_KEYRING_ACTION") {
	case "set":
		err = k.Set("service", "user", "second-token")
	case "delete":
		err = k.Delete("service", "user")
	case "get":
		_, err = k.Get("service", "user")
	default:
		t.Fatal("未知の操作")
	}
	if os.Getenv("WIKINO_TEST_KEYRING_EXPECT") == "timeout" {
		if !errors.Is(err, errKeyringTimeout) {
			t.Fatalf("ロック待ちの時間切れを期待: %v", err)
		}
	} else if err != nil {
		t.Fatal(err)
	}
}

// execKeyringは、go-keyringのmacOS実装と同じく、別のプロセスでOSの保存先に書き込む。
// 呼び出したCLIのプロセスが終了しても、書き込みは続く。releaseのファイルができるまで (最長10秒) 書き込みを待つ。
type execKeyring struct {
	fileKeyring
	release string
}

func (k execKeyring) Set(_, _, value string) error {
	script := `i=0; while [ ! -e "$3" ] && [ $i -lt 500 ]; do sleep 0.02; i=$((i+1)); done; printf %s "$1" > "$2"`
	return exec.Command("sh", "-c", script, "sh", value, k.path, k.release).Run() // #nosec G204 -- テストで作った値と一時ファイルだけを渡す
}

func TestStoreDetectsLateSetAfterProcessExit(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("shで遅れた書き込みを再現する")
	}
	dir := t.TempDir()
	run := func(action string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestStoreLoginProcess$")
		cmd.Env = append(os.Environ(), "WIKINO_TEST_KEYRING_DIR="+dir, "WIKINO_TEST_KEYRING_ACTION="+action)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("子プロセス (%s) = %v\n%s", action, err, output)
		}
	}
	// プロセスAはキーチェーンの時間切れでファイルに保存して終了し、ロックを手放す。
	run("slow-login")
	// プロセスBは再ログインでキーチェーンに保存する。
	run("login")
	// プロセスBの成功の後で、プロセスAの遅れたSetを完了させる。
	if err := os.WriteFile(filepath.Join(dir, "release"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "token")
	deadline := time.Now().Add(5 * time.Second)
	for {
		data, _ := os.ReadFile(path) // #nosec G304 -- テストの一時ファイル
		if strings.Contains(string(data), "first-token") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("プロセスAの遅れたSetが完了しない: %q", data)
		}
		time.Sleep(20 * time.Millisecond)
	}
	cfg, err := config.Load(filepath.Join(dir, "wikino"))
	if err != nil {
		t.Fatal(err)
	}
	store, _ := newTestStore(t, fileKeyring{path: path}, nil)
	store.Dir = filepath.Join(dir, "wikino")
	cred := &cfg.Credentials[0]
	if cred.Store != StoreKeyring {
		t.Fatalf("再ログインの保存先 = %+v", cred)
	}
	// 遅れたSetで置き換わったトークンを、再ログインのトークンとして使わない。
	assertNotLoggedIn(t, func() error { _, err := store.Load(cred); return err }())
}

// TestStoreLoginProcessは、TestStoreDetectsLateSetAfterProcessExitから子プロセスとして実行する。
func TestStoreLoginProcess(t *testing.T) {
	t.Parallel()
	dir := os.Getenv("WIKINO_TEST_KEYRING_DIR")
	if dir == "" {
		t.Skip("親テストから子プロセスとして実行する")
	}
	path := filepath.Join(dir, "token")
	var backend Keyring = fileKeyring{path: path}
	token := "second-token"
	if os.Getenv("WIKINO_TEST_KEYRING_ACTION") == "slow-login" {
		backend, token = execKeyring{fileKeyring: fileKeyring{path: path}, release: filepath.Join(dir, "release")}, "first-token"
	}
	store, _ := newTestStore(t, &systemKeyring{backend: backend, timeout: 100 * time.Millisecond, getenv: keyringTestEnv(dir)}, nil)
	store.Dir = filepath.Join(dir, "wikino")
	if err := config.WithLocked(context.Background(), store.Dir, func(cfg *config.Config) error {
		cred := testCredential()
		var err error
		if cred.Store, err = store.Save(cred, Secret{AccessToken: token}); err != nil {
			return err
		}
		cfg.AddCredential(*cred)
		return cfg.Save(store.Dir)
	}); err != nil {
		t.Fatal(err)
	}
}

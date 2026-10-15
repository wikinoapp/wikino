package auth

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// delayedKeyringは、Set/Getの応答を止め、タイムアウト後に完了させるためのキーチェーン。
type delayedKeyring struct {
	mu       sync.Mutex
	value    string
	started  chan struct{}
	release  chan struct{}
	finished chan struct{}
}

func (k *delayedKeyring) Get(_, _ string) (string, error) {
	close(k.started)
	<-k.release
	defer close(k.finished)
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.value, nil
}
func (k *delayedKeyring) Set(_, _, value string) error {
	close(k.started)
	<-k.release
	defer close(k.finished)
	k.mu.Lock()
	defer k.mu.Unlock()
	k.value = value
	return nil
}
func (k *delayedKeyring) Delete(_, _ string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.value = ""
	return nil
}

func TestSystemKeyringGetTimeout(t *testing.T) {
	t.Parallel()
	backend := &delayedKeyring{value: testToken, started: make(chan struct{}), release: make(chan struct{}), finished: make(chan struct{})}
	k := &systemKeyring{backend: backend, timeout: 30 * time.Millisecond, getenv: keyringTestEnv(t.TempDir())}
	value, err := k.Get("service", "user")
	if value != "" || !errors.Is(err, errKeyringTimeout) {
		t.Fatalf("タイムアウト時の取得 = %q, %v", value, err)
	}
	// 時間切れの後に取得を完了させ、共有した変数の読み書きが無いことを `-race` で確かめる。
	<-backend.started
	close(backend.release)
	<-backend.finished
}

func TestSystemKeyringDeletesAfterLateSet(t *testing.T) {
	t.Parallel()
	backend := &delayedKeyring{started: make(chan struct{}), release: make(chan struct{}), finished: make(chan struct{})}
	k := &systemKeyring{backend: backend, timeout: 30 * time.Millisecond, getenv: keyringTestEnv(t.TempDir())}
	if err := k.Set("service", "user", testToken); !errors.Is(err, errKeyringTimeout) {
		t.Fatalf("保存のエラー = %v、時間切れを期待", err)
	}
	<-backend.started
	// 実行中のSetが終わらないうちは、削除を成功扱いにしない。
	if err := k.Delete("service", "user"); !errors.Is(err, errKeyringTimeout) {
		t.Fatalf("削除のエラー = %v、時間切れを期待", err)
	}
	close(backend.release)
	<-backend.finished
	if err := k.Delete("service", "user"); err != nil {
		t.Fatal(err)
	}
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.value != "" {
		t.Error("時間切れ後のSetが削除を追い越した")
	}
}

func TestSystemKeyringGetSuccess(t *testing.T) {
	t.Parallel()
	backend := newFakeKeyring()
	if err := backend.Set("service", "user", testToken); err != nil {
		t.Fatal(err)
	}
	k := &systemKeyring{backend: backend, timeout: time.Second, getenv: keyringTestEnv(t.TempDir())}
	value, err := k.Get("service", "user")
	if value != testToken || err != nil {
		t.Fatalf("取得 = %q, %v", value, err)
	}
}

func TestSystemKeyringSkipsExpiredQueuedSet(t *testing.T) {
	t.Parallel()
	backend := &delayedKeyring{started: make(chan struct{}), release: make(chan struct{}), finished: make(chan struct{})}
	k := &systemKeyring{backend: backend, timeout: 30 * time.Millisecond, getenv: keyringTestEnv(t.TempDir())}
	if _, err := k.Get("service", "user"); !errors.Is(err, errKeyringTimeout) {
		t.Fatalf("取得のエラー = %v、時間切れを期待", err)
	}
	<-backend.started
	if err := k.Set("service", "user", testToken); !errors.Is(err, errKeyringTimeout) {
		t.Fatalf("待機中の保存のエラー = %v、時間切れを期待", err)
	}
	close(backend.release)
	<-backend.finished
	// 未開始のSetが実行されると、backendはstartedを再度closeして失敗する。
	if err := k.Delete("service", "user"); err != nil {
		t.Fatal(err)
	}
}

// keyringTestEnvは、テストごとのロックを一時ディレクトリへ置くために使う。
func keyringTestEnv(dir string) func(string) string {
	return func(name string) string {
		if name == "XDG_CONFIG_HOME" {
			return dir
		}
		return ""
	}
}

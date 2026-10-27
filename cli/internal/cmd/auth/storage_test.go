package auth

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	tokenauth "github.com/wikinoapp/wikino/cli/internal/auth"
	"github.com/wikinoapp/wikino/cli/internal/config"
	"github.com/wikinoapp/wikino/cli/internal/iostreams"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

func TestLoginRestoresOnConfigSaveFailure(t *testing.T) {
	t.Parallel()
	tests := []struct{ name, oldStore, newStore string }{
		{name: "新規キーチェーン", newStore: "keyring"},
		{name: "新規ファイル", newStore: "file"},
		{name: "キーチェーンの上書き", oldStore: "keyring", newStore: "keyring"},
		{name: "ファイルの上書き", oldStore: "file", newStore: "file"},
		{name: "キーチェーンからファイル", oldStore: "keyring", newStore: "file"},
		{name: "ファイルからキーチェーン", oldStore: "file", newStore: "keyring"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := newTestEnv(t)
			e.ios, e.stdin, e.stdout, e.stderr = iostreams.Test()
			if tt.oldStore != "" {
				e.vars[tokenauth.EnvTokenStore] = tt.oldStore
				e.login("team", aliceToken)
			}
			e.vars[tokenauth.EnvTokenStore] = tt.newStore
			f := e.factory()
			dir, err := f.ConfigDir()
			if err != nil {
				t.Fatal(err)
			}
			cred := config.Credential{Host: config.DefaultHost, Space: "team", MemberID: testOwners[aliceToken].memberID, MemberAtname: "alice"}
			var old *config.Credential
			cfgPath := filepath.Join(dir, "config.toml")
			err = config.WithLocked(context.Background(), dir, func(cfg *config.Config) error {
				if len(cfg.Credentials) > 0 {
					copy := cfg.Credentials[0]
					old = &copy
					if err := os.Rename(cfgPath, cfgPath+".backup"); err != nil {
						return err
					}
				}
				// 設定を読み終えてから保存先をディレクトリにし、最後のrenameだけを失敗させる。
				if err := os.Mkdir(cfgPath, 0700); err != nil {
					return err
				}
				_, _, err := saveLogin(f, dir, cfg, cred, tokenauth.Secret{AccessToken: "replacement-secret"})
				return err
			})
			if err == nil {
				t.Fatal("設定保存失敗を期待")
			}
			if err := os.Remove(cfgPath); err != nil {
				t.Fatal(err)
			}
			if old != nil {
				if err := os.Rename(cfgPath+".backup", cfgPath); err != nil {
					t.Fatal(err)
				}
				secret, err := f.TokenStore(dir).Load(old)
				if err != nil || secret.AccessToken != aliceToken {
					t.Fatalf("復元したトークン = %q, %v", secret.AccessToken, err)
				}
			}
			// 新しく追加した保存先にはトークンを残さない。
			if old == nil || old.Store != tt.newStore {
				switch tt.newStore {
				case tokenauth.StoreKeyring:
					if len(e.keyring.items) != 0 {
						t.Fatal("新しい保存先のキーチェーンにトークンが残った")
					}
				case tokenauth.StoreFile:
					files, err := os.ReadDir(filepath.Join(dir, "credentials"))
					if err != nil && !errors.Is(err, os.ErrNotExist) {
						t.Fatal(err)
					}
					if len(files) != 0 {
						t.Fatal("新しい保存先にトークンのファイルが残った")
					}
				}
			}
			e.assertNoSecret()
		})
	}
}

func TestLoginReportsRestoreFailure(t *testing.T) {
	t.Parallel()
	e := newTestEnv(t)
	e.login("team", aliceToken)
	f := e.factory()
	dir, err := f.ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, "config.toml")
	calls := 0
	e.keyring.beforeSet = func() {
		calls++
		if calls == 2 {
			e.keyring.setErr = errors.New("復元の保存失敗")
		}
	}
	err = config.WithLocked(context.Background(), dir, func(cfg *config.Config) error {
		cred := cfg.Credentials[0]
		if err := os.Rename(cfgPath, cfgPath+".backup"); err != nil {
			return err
		}
		if err := os.Mkdir(cfgPath, 0700); err != nil {
			return err
		}
		_, _, err := saveLogin(f, dir, cfg, cred, tokenauth.Secret{AccessToken: "replacement-secret"})
		return err
	})
	cliErr := assertCLIError(t, err, output.CodeUnexpected, output.ExitError)
	if !strings.Contains(cliErr.Message, "復元または削除にも失敗") {
		t.Fatalf("復元失敗が通知されない: %s", cliErr.Message)
	}
	if strings.Contains(cliErr.Message+cliErr.Hint, "replacement-secret") {
		t.Error("エラーに秘密を含む")
	}
}

func TestLoginCleanupFailureCanBeRetried(t *testing.T) {
	t.Parallel()
	for _, retry := range []string{"logout", "login"} {
		t.Run(retry, func(t *testing.T) {
			t.Parallel()
			e := newTestEnv(t)
			e.login("team", aliceToken)
			e.keyring.setErr = errors.New("保存失敗")
			e.keyring.deleteErr = errors.New("削除失敗")
			// 新しい資格情報は保存済みのため、旧保存先の削除だけの失敗ではログインを成功にして警告する。
			if err := e.run(aliceToken, "auth", "login", "team", "--with-token"); err != nil {
				t.Fatalf("旧保存先の削除の失敗でログインが失敗した: %v", err)
			}
			if !strings.Contains(e.stderr.String(), "以前の保存先からトークンを削除できませんでした") {
				t.Errorf("削除の失敗の警告がない: %q", e.stderr.String())
			}
			if !strings.Contains(e.stdout.String(), "alice") {
				t.Errorf("保存した資格情報が出力されない: %q", e.stdout.String())
			}
			cfg := e.config()
			if cfg.Credentials[0].Store != "file" || cfg.Credentials[0].CleanupStore != "keyring" {
				t.Fatalf("削除再試行の情報がない: %+v", cfg.Credentials[0])
			}
			if len(e.keyring.items) != 1 {
				t.Fatal("失敗した削除で旧トークンが消えた")
			}
			// 削除の失敗中にlogoutしても、資格情報と再試行情報を残す。
			err := e.run("", "auth", "logout", "team")
			_ = assertCLIError(t, err, output.CodeUnexpected, output.ExitError)
			if e.config().Credentials[0].CleanupStore != "keyring" {
				t.Fatal("失敗したlogoutで再試行情報が消えた")
			}
			e.keyring.deleteErr = nil
			if retry == "logout" {
				if err := e.run("", "auth", "logout", "team"); err != nil {
					t.Fatal(err)
				}
				if len(e.config().Credentials) != 0 {
					t.Fatal("logout後に設定が残った")
				}
			} else {
				e.vars[tokenauth.EnvTokenStore] = "file"
				if err := e.run(aliceToken, "auth", "login", "team", "--with-token"); err != nil {
					t.Fatal(err)
				}
				if e.config().Credentials[0].CleanupStore != "" {
					t.Fatal("削除成功後に再試行情報が残った")
				}
			}
			if len(e.keyring.items) != 0 {
				t.Fatal("再試行後に旧トークンが残った")
			}
			e.assertNoSecret()
		})
	}
}

func TestLoginKeepsUnreadablePreviousToken(t *testing.T) {
	t.Parallel()
	e := newTestEnv(t)
	e.login("team", aliceToken)
	e.keyring.err = errors.New("キーチェーンがロックされている")
	err := e.run(aliceToken, "auth", "login", "team", "--with-token")
	cliErr := assertCLIError(t, err, output.CodeUnexpected, output.ExitError)
	if !strings.Contains(cliErr.Message, "上書きを中止") {
		t.Fatalf("退避失敗を通知しない: %s", cliErr.Message)
	}
	if e.config().Credentials[0].Store != "keyring" || len(e.keyring.items) != 1 {
		t.Fatal("退避できないトークンが変更された")
	}
}

func TestConcurrentLoginsKeepAllCredentials(t *testing.T) {
	t.Parallel()
	sharedDir := t.TempDir()
	var ready sync.WaitGroup
	ready.Add(3)
	results := make(chan error, 3)
	for _, token := range []string{aliceToken, botToken, carolToken} {
		e := newTestEnv(t)
		e.vars["XDG_CONFIG_HOME"] = sharedDir
		e.vars[tokenauth.EnvTokenStore] = "file"
		// 全コマンドが最初の設定を読み終わってから、API検証を同時に完了させる。
		e.beforeResponse = func() { ready.Done(); ready.Wait() }
		go func() { results <- e.run(token, "auth", "login", testOwners[token].space, "--with-token") }()
	}
	for range 3 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	dir := filepath.Join(sharedDir, "wikino")
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Credentials) != 3 {
		t.Fatalf("同時ログイン後の資格情報数 = %d、期待値 = 3", len(cfg.Credentials))
	}
	f := newTestEnv(t)
	f.ios, f.stdin, f.stdout, f.stderr = iostreams.Test()
	for _, cred := range cfg.Credentials {
		secret, err := f.factory().TokenStore(dir).Load(&cred)
		if err != nil || secret.AccessToken == "" {
			t.Fatalf("資格情報のトークンを取得できない: %v", err)
		}
	}
}

func TestLoginRestoresKeyringAfterFailedStoreSwitch(t *testing.T) {
	t.Parallel()
	e := newTestEnv(t)
	e.login("team", aliceToken)
	f := e.factory()
	dir, err := f.ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, "config.toml")
	calls := 0
	e.keyring.beforeSet = func() {
		calls++
		if calls == 1 {
			// 保存が反映された後にエラーになり、ファイルへ切り替わった状況を再現する。
			for key := range e.keyring.items {
				e.keyring.items[key] = `{"access_token":"replacement-secret"}`
			}
			e.keyring.setErr = errors.New("保存後の応答失敗")
		} else {
			e.keyring.setErr = nil
		}
	}
	var old config.Credential
	err = config.WithLocked(context.Background(), dir, func(cfg *config.Config) error {
		old = cfg.Credentials[0]
		if err := os.Rename(cfgPath, cfgPath+".backup"); err != nil {
			return err
		}
		if err := os.Mkdir(cfgPath, 0700); err != nil {
			return err
		}
		_, _, err := saveLogin(f, dir, cfg, old, tokenauth.Secret{AccessToken: "replacement-secret"})
		return err
	})
	if err == nil {
		t.Fatal("設定保存失敗を期待")
	}
	secret, err := f.TokenStore(dir).Load(&old)
	if err != nil || secret.AccessToken != aliceToken {
		t.Fatalf("元のトークンが復元されない: %v", err)
	}
}

func TestInvalidStoreDoesNotRestoreOrDeletePendingCredential(t *testing.T) {
	t.Parallel()
	e := newTestEnv(t)
	e.vars[tokenauth.EnvTokenStore] = "file"
	e.login("team", aliceToken)
	f := e.factory()
	dir, err := f.ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	err = config.WithLocked(context.Background(), dir, func(cfg *config.Config) error {
		cfg.Credentials[0].CleanupStore = "keyring"
		return cfg.Save(dir)
	})
	if err != nil {
		t.Fatal(err)
	}
	e.keyring.err = errors.New("キーチェーン利用不可")
	e.vars[tokenauth.EnvTokenStore] = "invalid"
	err = e.run(aliceToken, "auth", "login", "team", "--with-token")
	_ = assertCLIError(t, err, output.CodeUsage, output.ExitUsage)
	cred := e.config().Credentials[0]
	secret, err := f.TokenStore(dir).Load(&cred)
	if err != nil || secret.AccessToken != aliceToken || cred.CleanupStore != "keyring" {
		t.Fatal("使い方の誤りで保存済み資格情報が変更された")
	}
}

func TestLoginRollbackRemovesTimedOutStore(t *testing.T) {
	t.Parallel()
	e := newTestEnv(t)
	e.vars[tokenauth.EnvTokenStore] = "file"
	e.login("team", aliceToken)
	f := e.factory()
	dir, err := f.ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	store := f.TokenStore(dir)
	previous := e.config().Credentials[0]
	// saveLoginと同じく、上書きする前に元のトークンを世代ごと退避する。
	backup, err := store.Load(&previous)
	if err != nil {
		t.Fatal(err)
	}
	// キーチェーンのSetが時間切れになってファイルへ切り替え、遅れてキーチェーンにも保存された状態。
	saved := previous
	saved.CleanupStore = tokenauth.StoreKeyring
	pending := saved
	pending.Store = tokenauth.StoreKeyring
	if err := store.Restore(&pending, tokenauth.Secret{AccessToken: "replacement-secret"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Restore(&saved, tokenauth.Secret{AccessToken: "replacement-secret"}); err != nil {
		t.Fatal(err)
	}

	if err := rollbackLogin(store, &previous, backup, saved); err != nil {
		t.Fatal(err)
	}

	if len(e.keyring.items) != 0 {
		t.Error("元の設定に記録の無い時間切れの保存先にトークンが残った")
	}
	secret, err := store.Load(&previous)
	if err != nil || secret.AccessToken != aliceToken {
		t.Fatalf("元のトークンが復元されない: %v", err)
	}
}

func TestLoginRollbackKeepsUntouchedPreviousToken(t *testing.T) {
	t.Parallel()
	e := newTestEnv(t)
	e.login("team", aliceToken)
	f := e.factory()
	dir, err := f.ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, "config.toml")
	// キーチェーンへの保存が反映されずに失敗し続け、ファイルへ切り替わった状況。
	e.keyring.setErr = errors.New("保存失敗")
	var old config.Credential
	err = config.WithLocked(context.Background(), dir, func(cfg *config.Config) error {
		old = cfg.Credentials[0]
		if err := os.Rename(cfgPath, cfgPath+".backup"); err != nil {
			return err
		}
		if err := os.Mkdir(cfgPath, 0700); err != nil {
			return err
		}
		_, _, err := saveLogin(f, dir, cfg, old, tokenauth.Secret{AccessToken: "replacement-secret"})
		return err
	})
	if err == nil {
		t.Fatal("設定保存失敗を期待")
	}
	// 元のトークンは変わっていないため、使えないキーチェーンへ書き直さず、復元の失敗も通知しない。
	var cliErr *output.Error
	if errors.As(err, &cliErr) {
		t.Fatalf("復元の失敗を通知した: %s", cliErr.Message)
	}
	secret, err := f.TokenStore(dir).Load(&old)
	if err != nil || secret.AccessToken != aliceToken {
		t.Fatalf("元のトークンが変わった: %v", err)
	}
}

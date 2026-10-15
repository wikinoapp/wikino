// Package authは、APIを呼ぶためのトークンの保存と取り出しを扱う。
//
// トークンはOSのキーチェーンに保存し、使えない環境 (開発コンテナなど) では0600のファイルに保存する。
// どこに保存したかは設定ファイルの資格情報 (config.Credential) に記録する。
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/wikinoapp/wikino/cli/internal/apiclient"
	"github.com/wikinoapp/wikino/cli/internal/config"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// EnvTokenStoreは、トークンの保存先を選ぶ環境変数。`file` でファイルへの保存を強制する。
const EnvTokenStore = "WIKINO_TOKEN_STORE"

// トークンの保存先。設定ファイルの資格情報のstoreに記録する値で、変えない。
const (
	StoreKeyring = "keyring"
	StoreFile    = "file"
)

// keyringServiceは、キーチェーンの項目のサービス名。
const keyringService = "wikino-cli"

// keyringTimeoutは、キーチェーンの操作を待つ時間の上限。
// LinuxのSecret Serviceは、デーモンの応答が無いと戻らないことがあるため打ち切る。
const keyringTimeout = 3 * time.Second

// credentialsDirNameは、ファイルに保存するトークンを置く、設定のディレクトリの中のディレクトリ。
const credentialsDirName = "credentials"

// Secretは保存するトークン。
// 個人アクセストークンはAccessTokenだけを持ち、OAuthのトークンはリフレッシュトークンと期限も持つ。
type Secret struct {
	AccessToken string `json:"access_token"`
	// RefreshTokenはOAuthのリフレッシュトークン。個人アクセストークンでは空。
	RefreshToken string `json:"refresh_token,omitempty"`
	// ExpiresAtはOAuthのアクセストークンの期限。
	ExpiresAt time.Time `json:"expires_at,omitzero"`
	// ScopesはOAuthのトークンの応答の `scope` が示す、付与されたスコープ。
	Scopes []string `json:"scopes,omitempty"`
	// Generationは、保存したときの資格情報の世代 (config.CredentialのGeneration)。
	Generation string `json:"generation"`
}

// IsOAuthは、OAuthで得たトークン (リフレッシュで更新するもの) かを返す。
func (s Secret) IsOAuth() bool {
	return s.RefreshToken != ""
}

// Keyringは、OSのキーチェーンの読み書き。テストで差し替える。
type Keyring interface {
	Get(service, user string) (string, error)
	Set(service, user, password string) error
	Delete(service, user string) error
}

// ErrKeyringNotFoundは、キーチェーンに項目が無いことを表す。Keyringの実装はこれを返す。
var ErrKeyringNotFound = keyring.ErrNotFound

// errKeyringTimeoutは、操作が時間内に終わらず、後から完了する可能性があることを表す。
var errKeyringTimeout = errors.New("キーチェーンの応答がありません")

// SystemKeyringは、OSのキーチェーン (macOS Keychain・Windows Credential Manager・
// LinuxのSecret Service) を使うKeyringを返す。
// getenvで決めた設定ディレクトリにロックを置き、同じ設定を使うプロセス間で操作を排他する。
func SystemKeyring(getenv func(string) string) Keyring {
	return &systemKeyring{backend: nativeKeyring{}, timeout: keyringTimeout, getenv: getenv}
}

// nativeKeyringはライブラリへの呼び出しをまとめ、待ち時間の制御をsystemKeyringに任せる。
type nativeKeyring struct{}

func (nativeKeyring) Get(service, user string) (string, error) { return keyring.Get(service, user) }
func (nativeKeyring) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}
func (nativeKeyring) Delete(service, user string) error { return keyring.Delete(service, user) }

// systemKeyringは、資格情報ごとのプロセス間ロックでキーチェーン操作を直列化する。
// CLIのプロセスが生きている間は、時間切れ後も開始済みの操作がロックを保持する。
// プロセス終了後の遅延書き込みは、Store.Loadの世代検証で検出する。
// 設定を更新する呼び出し側は設定ロックを先に取り、この層から設定ロックは取得しない。
type systemKeyring struct {
	backend Keyring
	timeout time.Duration
	getenv  func(string) string
}

func (k *systemKeyring) Get(service, user string) (string, error) {
	return k.call(service, user, func() (string, error) { return k.backend.Get(service, user) })
}
func (k *systemKeyring) Set(service, user, password string) error {
	_, err := k.call(service, user, func() (string, error) { return "", k.backend.Set(service, user, password) })
	return err
}
func (k *systemKeyring) Delete(service, user string) error {
	_, err := k.call(service, user, func() (string, error) { return "", k.backend.Delete(service, user) })
	return err
}

func (k *systemKeyring) call(service, user string, fn func() (string, error)) (string, error) {
	dir, err := config.Dir(k.getenv)
	if err != nil {
		return "", err
	}
	// 項目名をパスとして扱わず、同じ設定を使うプロセスが同じ項目のロックを共有する。
	sum := sha256.Sum256([]byte(service + "\x00" + user))
	name := hex.EncodeToString(sum[:]) + ".lock"
	ctx, cancel := context.WithTimeout(context.Background(), k.timeout)
	defer cancel()
	type result struct {
		value string
		err   error
	}
	done := make(chan result, 1)
	go func() {
		var value string
		err := config.WithFileLock(ctx, filepath.Join(dir, "keyring-locks"), name, func() error {
			var err error
			value, err = fn()
			return err
		})
		done <- result{value: value, err: err}
	}()
	select {
	case r := <-done:
		if errors.Is(r.err, context.DeadlineExceeded) || errors.Is(r.err, context.Canceled) {
			return "", errKeyringTimeout
		}
		return r.value, r.err
	case <-ctx.Done():
		return "", errKeyringTimeout
	}
}

// Storeは、資格情報ごとのトークンを保存し、取り出す。
type Store struct {
	// Keyringはキーチェーン。nilならキーチェーンを使わず、ファイルに保存する。
	Keyring Keyring
	// Dirは設定のディレクトリ。
	Dir    string
	Getenv func(string) string
	// Warnは警告 (キーチェーンを使えなかったこと・パーミッションが緩いこと) の出力先 (標準エラー)。
	Warn io.Writer
	Loc  *i18n.Localizer
	// OAuthは、OAuthのトークンをリフレッシュするときに認可サーバーと通信するクライアントを作る。
	// 接続先は資格情報ごとに違うため、接続先を受け取って作る。
	OAuth func(host string) *OAuthClient
	// Nowは現在の時刻を返す。nilならtime.Now。テストで差し替える。
	Now func() time.Time
}

// Saveは、credのトークンを保存し、保存先 (StoreKeyringかStoreFile) を返す。
// キーチェーンに保存できなければファイルに保存し、保存できたら警告を出す。
// 保存のたびに新しい世代をcredとトークンの両方に入れる。
func (s *Store) Save(cred *config.Credential, secret Secret) (string, error) {
	cred.Generation = rand.Text()
	secret.Generation = cred.Generation
	data, err := json.Marshal(secret) // #nosec G117 -- 保存先に書くためのトークンで、出力には使わない
	if err != nil {
		return "", err
	}

	useKeyring, err := s.useKeyring()
	if err != nil {
		return "", err
	}
	fellBack := false
	if useKeyring {
		kerr := s.Keyring.Set(keyringService, key(cred), string(data))
		if kerr == nil {
			return StoreKeyring, nil
		}
		if errors.Is(kerr, errKeyringTimeout) {
			// 時間切れでもSet自体は後から完了しうるため、削除対象を記録する。
			cred.CleanupStore = StoreKeyring
		}
		fellBack = true
	}

	if err := config.WriteFileAtomic(s.credentialsDir(), fileName(cred), data); err != nil {
		return "", err
	}
	// 警告はファイルに保存したことを伝えるため、書き込みに失敗したときは出さない。
	if fellBack {
		s.warn(s.Loc.T("auth_warning_keyring_unavailable", map[string]any{"Path": s.filePath(cred)}))
	}

	return StoreFile, nil
}

// Loadは、credのトークンを取り出す。保存したトークンが見つからなければ、ログインし直しを促すエラーを返す。
func (s *Store) Load(cred *config.Credential) (Secret, error) {
	var data []byte

	switch cred.Store {
	case StoreKeyring:
		if s.Keyring == nil {
			return Secret{}, s.notFoundError(cred)
		}
		value, err := s.Keyring.Get(keyringService, key(cred))
		if errors.Is(err, ErrKeyringNotFound) {
			return Secret{}, s.notFoundError(cred)
		}
		if err != nil {
			return Secret{}, &output.Error{
				Code:     output.CodeUnexpected,
				Message:  s.Loc.T("error_keyring_read"),
				Hint:     s.Loc.T("error_keyring_read_hint"),
				ExitCode: output.ExitError,
				Err:      err,
			}
		}
		data = []byte(value)
	default:
		s.warnLoosePermissions(s.Dir, s.credentialsDir(), s.filePath(cred))
		var err error
		data, err = os.ReadFile(s.filePath(cred))
		if errors.Is(err, fs.ErrNotExist) {
			return Secret{}, s.notFoundError(cred)
		}
		if err != nil {
			return Secret{}, err
		}
	}

	var secret Secret
	if err := json.Unmarshal(data, &secret); err != nil || secret.AccessToken == "" {
		return Secret{}, s.notFoundError(cred)
	}
	// 時間切れのキーチェーン操作は、CLIのプロセスが終わった後にOSの側で完了しうる
	// (macOSの `security` の子プロセスなど)。プロセス間ロックでは防げないため、
	// 後の再ログインで保存したトークンを古いトークンが上書きしたことを世代の違いで検出し、
	// 古いトークンを黙って使わずにログインし直しを促す。
	if secret.Generation != cred.Generation {
		return Secret{}, s.notFoundError(cred)
	}

	return secret, nil
}

// Deleteは、現在と旧保存先のトークンを消す。既に無ければ何もしない。
// どれかの保存先で削除に失敗した場合はエラーを返す。
func (s *Store) Delete(cred *config.Credential) error {
	err := s.deleteFrom(cred, cred.Store)
	if cred.CleanupStore != "" && cred.CleanupStore != cred.Store {
		err = errors.Join(err, s.deleteFrom(cred, cred.CleanupStore))
	}
	return err
}

// Cleanupは、以前の保存先だけを削除する。失敗した場合は再試行できるようCleanupStoreを残す。
func (s *Store) Cleanup(cred *config.Credential) error {
	if cred.CleanupStore == "" {
		return nil
	}
	if cred.CleanupStore != cred.Store {
		if err := s.deleteFrom(cred, cred.CleanupStore); err != nil {
			return err
		}
	}
	cred.CleanupStore = ""
	return nil
}

// Replaceは、credの今の保存先と世代のまま、トークンを置き換える。
// リフレッシュで得たトークンを保存するために使い、設定ファイルは変えない。
func (s *Store) Replace(cred *config.Credential, secret Secret) error {
	secret.Generation = cred.Generation
	return s.Restore(cred, secret)
}

// Restoreは、トークンを指定された元の保存先へ復元する。別の保存先へのフォールバックはしない。
func (s *Store) Restore(cred *config.Credential, secret Secret) error {
	data, err := json.Marshal(secret) // #nosec G117 -- 復元するトークンで、出力には使わない
	if err != nil {
		return err
	}
	if cred.Store == StoreKeyring {
		if s.Keyring == nil {
			return errors.New("キーチェーンを使えません")
		}
		return s.Keyring.Set(keyringService, key(cred), string(data))
	}
	return config.WriteFileAtomic(s.credentialsDir(), fileName(cred), data)
}

func (s *Store) deleteFrom(cred *config.Credential, storage string) error {
	if storage == StoreKeyring {
		if s.Keyring == nil {
			return errors.New("キーチェーンを使えません")
		}
		err := s.Keyring.Delete(keyringService, key(cred))
		if err != nil && !errors.Is(err, ErrKeyringNotFound) {
			return fmt.Errorf("キーチェーンからトークンを消せません: %w", err)
		}

		return nil
	}

	err := os.Remove(s.filePath(cred))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	return nil
}

// TokenSourceは、targetの要求に付けるトークンを返す。WIKINO_TOKENがあればそれを使い、
// 無ければ保存したトークンを取り出す。
func (s *Store) TokenSource(target *config.Target) (apiclient.TokenSource, error) {
	if target.EnvToken != "" {
		return apiclient.StaticToken(target.EnvToken), nil
	}

	secret, err := s.Load(target.Credential)
	if err != nil {
		return nil, err
	}
	if secret.IsOAuth() {
		return &refreshingToken{store: s, cred: *target.Credential, secret: secret}, nil
	}

	return apiclient.StaticToken(secret.AccessToken), nil
}

func (s *Store) useKeyring() (bool, error) {
	switch value := s.Getenv(EnvTokenStore); value {
	case "", StoreKeyring:
		return s.Keyring != nil, nil
	case StoreFile:
		return false, nil
	default:
		return false, &output.Error{
			Code:     output.CodeUsage,
			Message:  s.Loc.T("usage_error_invalid_token_store", map[string]any{"Value": value}),
			Hint:     s.Loc.T("usage_error_invalid_token_store_hint"),
			ExitCode: output.ExitUsage,
		}
	}
}

func (s *Store) notFoundError(cred *config.Credential) *output.Error {
	return &output.Error{
		Code:     output.CodeNotLoggedIn,
		Message:  s.Loc.T("error_token_not_found", map[string]any{"Space": cred.Space, "Member": cred.MemberAtname}),
		Hint:     s.Loc.T("error_space_not_logged_in_hint", map[string]any{"Space": cred.Space}),
		ExitCode: output.ExitAuth,
	}
}

// warnLoosePermissionsは、設定のディレクトリ・トークンのディレクトリ・トークンのファイルを本人以外が読み書きできるときに警告する。
// Windowsはパーミッションのビットでアクセスを制御しないため調べない。
func (s *Store) warnLoosePermissions(paths ...string) {
	if runtime.GOOS == "windows" {
		return
	}

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm()&0o077 == 0 {
			continue
		}

		mode := "600"
		if info.IsDir() {
			mode = "700"
		}
		s.warn(s.Loc.T("auth_warning_loose_permissions", map[string]any{
			"Path": path, "Perm": fmt.Sprintf("%o", info.Mode().Perm()), "Mode": mode,
		}))
	}
}

func (s *Store) warn(message string) {
	if s.Warn != nil {
		_, _ = fmt.Fprintln(s.Warn, message)
	}
}

func (s *Store) credentialsDir() string {
	return filepath.Join(s.Dir, credentialsDirName)
}

func (s *Store) filePath(cred *config.Credential) string {
	return filepath.Join(s.credentialsDir(), fileName(cred))
}

// keyは、資格情報の組をキーチェーンの項目のアカウント名にする。
// スペースの識別子は大文字と小文字を区別しないため、小文字にそろえる。
func key(cred *config.Credential) string {
	return cred.Host + " " + strings.ToLower(cred.Space) + " " + cred.MemberID
}

// fileNameは、資格情報の組からファイル名を作る。接続先のURLの記号をファイル名に使わずに済むよう、
// キーのハッシュにする。
func fileName(cred *config.Credential) string {
	sum := sha256.Sum256([]byte(key(cred)))

	return hex.EncodeToString(sum[:16]) + ".json"
}

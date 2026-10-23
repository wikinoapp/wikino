package auth

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/wikinoapp/wikino/cli/internal/config"
)

// refreshMarginは、アクセストークンの期限のどれだけ前から更新するか。
// 要求の途中で期限が切れないよう、期限の少し前に、要求を送る前に更新する。
const refreshMargin = time.Minute

// credentialLocksDirNameは、資格情報ごとのロックを置く、設定のディレクトリの中のディレクトリ。
const credentialLocksDirName = "credential-locks"

// WithCredentialLockは、credの資格情報ごとのプロセス間ロックを取り、fnの完了まで保持する。
// リフレッシュトークンはローテーションされ、使用済みの再提示で許可ごと失効するため、
// 同じ資格情報のリフレッシュ・ログイン・ログアウトをこのロックで直列化する。
// ロックの順は設定ロック → 資格情報のロック → キーチェーンのロックとし、逆の順に取らない。
func WithCredentialLock(ctx context.Context, dir string, cred *config.Credential, fn func() error) error {
	name := strings.TrimSuffix(fileName(cred), ".json") + ".lock"

	return config.WithFileLock(ctx, filepath.Join(dir, credentialLocksDirName), name, fn)
}

// freshは、secretのアクセストークンをnowの時点で更新せずに使えるかを返す。
func (s Secret) fresh(now time.Time) bool {
	return s.ExpiresAt.After(now.Add(refreshMargin))
}

// NewSecretは、トークンエンドポイントの応答を受け取った時刻nowで、保存するトークンにする。
func NewSecret(token *TokenResponse, now time.Time) Secret {
	return Secret{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.expiresAt(now),
		Scopes:       token.scopes(),
	}
}

// refreshingTokenは、OAuthのアクセストークンを期限の前に更新して返すTokenSource。
type refreshingToken struct {
	store  *Store
	cred   config.Credential
	secret Secret
}

func (t *refreshingToken) Token(ctx context.Context) (string, error) {
	if t.secret.fresh(t.store.now()) {
		return t.secret.AccessToken, nil
	}

	locked := false
	err := WithCredentialLock(ctx, t.store.Dir, &t.cred, func() error {
		locked = true
		// 待つ間に他のプロセスが更新・再ログイン・ログアウトしていれば、その結果を使う。
		// 古いリフレッシュトークンを再提示すると、許可のトークンがすべて失効する。
		cred, err := t.store.reloadCredential(&t.cred)
		if err != nil {
			return err
		}
		secret, err := t.store.Load(cred)
		if err != nil {
			return err
		}
		if secret.fresh(t.store.now()) || !secret.IsOAuth() {
			t.cred, t.secret = *cred, secret
			return nil
		}

		client := t.store.OAuth(cred.Host)
		metadata, err := client.FetchMetadata(ctx)
		if err != nil {
			return t.store.convertRefreshError(cred, err)
		}
		token, err := client.Refresh(ctx, metadata, secret.RefreshToken, ResourceURL(cred.Host, cred.Space))
		if err != nil {
			return t.store.convertRefreshError(cred, err)
		}
		refreshed := NewSecret(token, t.store.now())
		// 使用済みのリフレッシュトークンは使えないため、保存に失敗しても今回の要求には新しいトークンを使う。
		// 次の実行では保存した古いリフレッシュトークンが拒否されるため、ログインし直しを促す。
		if err := t.store.Replace(cred, refreshed); err != nil {
			t.store.warn(t.store.Loc.T("auth_warning_refresh_save_failed", map[string]any{"Space": cred.Space}))
		}
		refreshed.Generation = cred.Generation
		t.cred, t.secret = *cred, refreshed
		return nil
	})
	// ロックを取った後の失敗 (要求の時間切れなど) は、変換済みのエラーをそのまま返す。
	// 時間切れのエラーもcontext.DeadlineExceededを包むため、ロック待ちの中止と取り違えないようにする。
	if err != nil && !locked {
		return "", t.store.convertLockError(err)
	}
	if err != nil {
		return "", err
	}

	return t.secret.AccessToken, nil
}

// reloadCredentialは、最新の設定からcredと同じ組の資格情報を読み直す。
// 既に消されていれば、ログインし直しを促すエラーを返す。
func (s *Store) reloadCredential(cred *config.Credential) (*config.Credential, error) {
	cfg, err := config.Load(s.Dir)
	if err != nil {
		return nil, config.ConvertError(s.Loc, err)
	}
	for _, latest := range cfg.FindCredentials(cred.Host, cred.Space) {
		if latest.MemberID == cred.MemberID {
			return latest, nil
		}
	}

	return nil, s.notFoundError(cred)
}

func (s *Store) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}

	return time.Now()
}

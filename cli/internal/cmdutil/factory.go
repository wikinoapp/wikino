package cmdutil

import (
	"context"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/apiclient"
	"github.com/wikinoapp/wikino/cli/internal/apiutil"
	"github.com/wikinoapp/wikino/cli/internal/auth"
	"github.com/wikinoapp/wikino/cli/internal/config"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/iostreams"
)

// Factoryは、コマンドが使う実行環境 (入出力・表示言語・環境変数・HTTPとキーチェーン) をまとめる。
// コマンドは標準出力や環境変数を直接触らずFactoryを経由し、テストではこれを差し替える。
type Factory struct {
	IOStreams *iostreams.IOStreams
	Localizer *i18n.Localizer
	// VersionはCLIのバージョン。
	Version string
	// Getenvは環境変数を引く。
	Getenv func(string) string
	// HTTPClientはAPIへの要求を送るクライアント。nilならapiclientの既定を使う。
	HTTPClient *http.Client
	// Keyringはトークンを保存するキーチェーン。nilならキーチェーンを使わずファイルに保存する。
	Keyring auth.Keyring
	// OpenBrowserは、ブラウザでURLを開く。nilならブラウザを開けないものとして、デバイス認可でログインする。
	OpenBrowser func(url string) error
	// Nowは現在の時刻を返す。nilならtime.Now。トークンの期限の判定をテストで差し替える。
	Now func() time.Time
	// Sleepは、ctxが終わらない限りdだけ待つ。nilなら実際に待つ。デバイス認可のポーリングの間隔をテストで差し替える。
	Sleep func(ctx context.Context, d time.Duration) error
	// Hostnameは端末のホスト名を返す。nilならos.Hostname。ログインで送る端末名の既定をテストで差し替える。
	Hostname func() (string, error)
}

// ConfigDirは設定のディレクトリを返す。
func (f *Factory) ConfigDir() (string, error) {
	return config.Dir(f.Getenv)
}

// TokenStoreは、dirにトークンを保存するStoreを返す。
func (f *Factory) TokenStore(dir string) *auth.Store {
	return &auth.Store{
		Keyring: f.Keyring,
		Dir:     dir,
		Getenv:  f.Getenv,
		Warn:    f.IOStreams.ErrOut,
		Loc:     f.Localizer,
		OAuth:   f.OAuthClient,
		Now:     f.Now,
	}
}

// OAuthClientは、hostの認可サーバーと通信するクライアントを返す。
func (f *Factory) OAuthClient(host string) *auth.OAuthClient {
	return &auth.OAuthClient{
		Host:       host,
		HTTPClient: f.HTTPClient,
		UserAgent:  apiutil.UserAgent(f.Version),
	}
}

// Resolverは、cmdのフラグとcfgで接続先・スペース・メンバーを決めるResolverを返す。
func (f *Factory) Resolver(cmd *cobra.Command, cfg *config.Config) *config.Resolver {
	return &config.Resolver{
		Config: cfg,
		Flags:  config.ReadFlags(cmd.Flags()),
		Getenv: f.Getenv,
		Loc:    f.Localizer,
	}
}

// NewAPIClientは、hostにtokenで要求を送るAPIのクライアントを作る。
func (f *Factory) NewAPIClient(host string, token apiclient.TokenSource) (*apiclient.Client, error) {
	return f.newAPIClient(host, token, nil)
}

func (f *Factory) newAPIClient(host string, token apiclient.TokenSource, dryRun *apiutil.DryRun) (*apiclient.Client, error) {
	var httpClient apiclient.HTTPDoer
	if f.HTTPClient != nil {
		httpClient = f.HTTPClient
	}
	return apiutil.NewClient(f.Localizer, apiutil.ClientOptions{
		Host:       host,
		Token:      token,
		Version:    f.Version,
		DryRun:     dryRun,
		HTTPClient: httpClient,
	})
}

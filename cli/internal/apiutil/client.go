// Package apiutilは、コマンドがAPIを呼ぶための部品を提供する。
//
// 公開APIのクライアント (apiclient) はCLIの他のパッケージに依存しないSDK相当の層のため、
// CLIに固有のこと (User-Agentと表示言語の決定、型付きのエラーからCLIのエラーとヒントへの変換、
// `--dry-run`) はこのパッケージで行う。
package apiutil

import (
	"fmt"
	"runtime"

	"github.com/wikinoapp/wikino/cli/internal/apiclient"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
)

// ClientOptionsは、CLIからAPIを呼ぶクライアントの設定。
type ClientOptions struct {
	// Hostは接続先のURL。
	Host string
	// Tokenは要求に付けるトークン。
	Token apiclient.TokenSource
	// VersionはCLIのバージョン (User-Agentに載せる)。
	Version string
	// DryRunがあれば、状態を変える要求を送らずにそこへ記録する (`--dry-run`)。
	DryRun *DryRun
	// HTTPClientは要求を送るクライアント。nilならapiclientの既定を使う。テストで差し替える。
	HTTPClient apiclient.HTTPDoer
}

// NewClientは、optsでAPIのクライアントを作る。
// 接続先が条件を満たさなければ、CLIのエラー (使い方の誤り) を返す。
func NewClient(loc *i18n.Localizer, opts ClientOptions) (*apiclient.Client, error) {
	clientOpts := apiclient.Options{
		Host:           opts.Host,
		Token:          opts.Token,
		UserAgent:      UserAgent(opts.Version),
		AcceptLanguage: loc.Lang(),
		HTTPClient:     opts.HTTPClient,
	}
	switch {
	case opts.DryRun == nil:
	case opts.DryRun.IncludeReadOnly:
		clientOpts.HTTPClient = opts.DryRun
	default:
		clientOpts.BeforeWrite = opts.DryRun.intercept
	}

	client, err := apiclient.New(clientOpts)
	if err != nil {
		return nil, ConvertError(loc, err)
	}

	return client, nil
}

// UserAgentは、APIへの要求に付けるUser-Agentを返す。
// サーバーのログで公式CLIの版と動作環境を見分けられるようにする。
func UserAgent(version string) string {
	return fmt.Sprintf("wikino-cli/%s (%s; %s)", version, runtime.GOOS, runtime.GOARCH)
}

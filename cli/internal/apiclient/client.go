// Package apiclientは、Wikinoの公開API (/api/v1) のクライアント。
//
// OpenAPI記述から生成したクライアント (apigen) に、トークンの付与・User-Agentと
// Accept-Languageの付与・Problem Detailsの型付きのエラー・カーソルのページ送り・
// 状態を変える要求の前のフックを足す。将来のデスクトップクライアントなどでも使えるよう、
// CLIの他のパッケージ (コマンド・出力・翻訳・設定) に依存しない (depguardで強制する)。
package apiclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/wikinoapp/wikino/cli/internal/apigen"
)

// basePathは、接続先のURLからAPIの基底までのパス。
const basePath = "/api/v1"

// DefaultTimeoutは、HTTPClientを指定しないときの1つの要求の時間の上限。
// 数百KBの本文の読み書きにも足り、応答の無い接続先でエージェントが止まり続けない長さにする。
const DefaultTimeout = 60 * time.Second

// TokenSourceは、要求に付けるアクセストークンを返す。
// トークンの更新 (リフレッシュ) は実装の側で行い、このパッケージは要求ごとに呼ぶだけにする。
type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

// StaticTokenは、常に同じトークンを返すTokenSource。個人アクセストークンに使う。
type StaticToken string

// TokenはトークンをそのままTokenSourceとして返す。
func (t StaticToken) Token(context.Context) (string, error) {
	return string(t), nil
}

// HTTPDoerは要求を送る。*http.Clientが満たす。
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Optionsはクライアントの設定。
type Options struct {
	// Hostは接続先のURL (例: https://wikino.app)。HTTPSか、ループバックのHTTPに限る。
	Host string
	// Tokenは要求に付けるトークン。nilならAuthorizationヘッダーを付けない。
	Token TokenSource
	// UserAgentはUser-Agentヘッダーの値。空なら付けない。
	UserAgent string
	// AcceptLanguageはAccept-Languageヘッダーの値。Problem Detailsの文言の言語を揃えるために送る。空なら付けない。
	AcceptLanguage string
	// HTTPClientは要求を送るクライアント。nilならNewHTTPClient(nil)の既定のクライアントを使う。
	// テストではここに差し替えたトランスポートを持つクライアントを渡す。
	HTTPClient HTTPDoer
	// BeforeWriteは、状態を変える要求 (操作の分類が読み取り専用でないもの) を送る直前に呼ぶ。
	// エラーを返すと要求を送らず、そのエラーを返す。`--dry-run` で送る代わりに要求を表示するために使う。
	// 受け取る要求にはAuthorizationヘッダーが付いているため、表示するときは値を伏せる。
	BeforeWrite func(req *http.Request) error
}

// ClientはWikinoの公開APIのクライアント。operationの呼び出しは生成したクライアントのメソッドで行う。
// 失敗の応答 (4xx・5xx) は、生成したクライアントの応答の型ではなく `*Error` として返る。
// 成功応答の読み取り・解析失敗は、送信済みと分かるよう `*apigen.ResponseError` として返る。
type Client struct {
	*apigen.ClientWithResponses

	// baseURLはAPIの基底のURL (末尾の "/" を含まない)。Doで任意のパスの要求を組み立てるために使う。
	baseURL string
	doer    HTTPDoer
	// httpClientは、トークンなどのヘッダーを付けずに要求を送るクライアント。署名付きURLからの取得 (Download) に使う。
	httpClient HTTPDoer
	userAgent  string
}

// Newは、optsのクライアントを作る。接続先が条件を満たさなければ `*InputError` を返す。
func New(opts Options) (*Client, error) {
	host, err := ValidateHost("host", opts.Host)
	if err != nil {
		return nil, err
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = NewHTTPClient(nil)
	}

	t := &transport{
		client:         httpClient,
		token:          opts.Token,
		userAgent:      opts.UserAgent,
		acceptLanguage: opts.AcceptLanguage,
		beforeWrite:    opts.BeforeWrite,
		basePath:       host.Path + basePath,
	}

	// 生成したクライアントはoperationのパスを "./spaces/..." として基底のURLに解決するため、
	// 基底のURLは "/" で終える。
	c, err := apigen.NewClientWithResponses(host.String()+basePath+"/", apigen.WithHTTPClient(t))
	if err != nil {
		return nil, err
	}

	return &Client{
		ClientWithResponses: c,
		baseURL:             host.String() + basePath,
		doer:                t,
		httpClient:          httpClient,
		userAgent:           opts.UserAgent,
	}, nil
}

// Doは、methodの要求をpath (APIの基底 `/api/v1` からのパス。クエリを含められる) へ送る。
// operationのメソッドが無いAPIを呼ぶためのもので、トークン・ヘッダーの付与、状態を変える要求の前のフック、
// 失敗の応答の `*Error` への変換は、生成したクライアントのメソッドと同じく行う。
// pathはParseAPIPathで検証し、受け付けなければ要求を送らずに `*InputError` を返す。
// 成功の応答の本文は呼び出し側が閉じる。
func (c *Client) Do(ctx context.Context, method, path string, body io.Reader, header http.Header) (*http.Response, error) {
	ref, err := ParseAPIPath("path", path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+ref.String(), body)
	if err != nil {
		return nil, err
	}
	for name, values := range header {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	return c.doer.Do(req)
}

// ErrInsecureRedirectは、HTTPSでもループバックのHTTPでもないリダイレクト先を辿らなかったことを表す。
// 最初の接続先には要求を送っているため、書き込みが反映された可能性がある。
var ErrInsecureRedirect = errors.New("apiclient: HTTPSでもループバックのHTTPでもないリダイレクト先への接続を拒否しました")

// NewHTTPClientは、HTTPClientを指定しないときの既定のクライアントを作る。
// DefaultTimeoutで打ち切り、接続先の条件を満たさないURLへのリダイレクトを辿らない。
// rtがnilならhttp.DefaultTransportを使う。テストではトランスポートだけを差し替え、
// 既定と同じリダイレクトの制限を通すために使う。
func NewHTTPClient(rt http.RoundTripper) *http.Client {
	return &http.Client{
		Transport:     rt,
		Timeout:       DefaultTimeout,
		CheckRedirect: checkRedirect,
	}
}

// checkRedirectは、接続先の条件 (HTTPSかループバック) を満たさないURLへのリダイレクトを辿らない。
// Goのクライアントは別のホストへのリダイレクトではAuthorizationを外すが、同じホストのHTTPへの
// リダイレクトでは付けたまま送るため、トークンが平文で流れないようにする。
func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("リダイレクトが10回に達したため中止しました")
	}
	if !isAllowedURLScheme(req.URL) {
		return ErrInsecureRedirect
	}

	return nil
}

// transportは、生成したクライアントの要求にヘッダーを付けて送り、失敗の応答を `*Error` にする。
type transport struct {
	client         HTTPDoer
	token          TokenSource
	userAgent      string
	acceptLanguage string
	beforeWrite    func(req *http.Request) error
	basePath       string
}

func (t *transport) Do(req *http.Request) (*http.Response, error) {
	// 呼び出し側 (Doのheader) が指定した値は上書きしない。
	if t.userAgent != "" && req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", t.userAgent)
	}
	if t.acceptLanguage != "" && req.Header.Get("Accept-Language") == "" {
		req.Header.Set("Accept-Language", t.acceptLanguage)
	}
	if t.token != nil {
		token, err := t.token.Token(req.Context())
		if err != nil {
			return nil, err
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	if t.beforeWrite != nil && !t.isReadOnly(req) {
		if err := t.beforeWrite(req); err != nil {
			return nil, err
		}
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		defer func() { _ = resp.Body.Close() }()
		return nil, newError(resp)
	}

	return resp, nil
}

// isReadOnlyは、reqが読み取り専用のoperationの要求かを返す。
// どのoperationにも当たらない要求は、状態を変えうるものとして扱う。
func (t *transport) isReadOnly(req *http.Request) bool {
	path, ok := strings.CutPrefix(req.URL.EscapedPath(), t.basePath)
	if !ok {
		return false
	}

	op, ok := MatchOperation(req.Method, path)

	return ok && op.ReadOnly
}

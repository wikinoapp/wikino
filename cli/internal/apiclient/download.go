package apiclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// ErrInsecureDownloadURLは、署名付きURLがHTTPSでもループバックのHTTPでもないため、要求を送らなかったことを表す。
var ErrInsecureDownloadURL = errors.New("apiclient: HTTPSでもループバックのHTTPでもない署名付きURLへの要求を拒否しました")

// DownloadErrorは、署名付きURLへの要求が成功 (2xx) 以外の応答を返したことを表す。
// 署名付きURLの置き場所 (オブジェクトストレージ) はProblem Detailsを返さないため、ステータスだけを持つ。
type DownloadError struct {
	// StatusCodeはHTTPステータスコード。
	StatusCode int
}

func (e *DownloadError) Error() string {
	return fmt.Sprintf("署名付きURLからの取得に失敗しました: HTTP %d", e.StatusCode)
}

// DownloadNetworkErrorは、署名付きURLへの通信の失敗 (応答の本文の読み取りの途中の失敗を含む) を表す。
// 原因には署名付きURLが含まれうるため、Errorは固定の安全な文言だけを返す。
type DownloadNetworkError struct {
	// Errは通信の失敗の原因。キャンセルやタイムアウトの判定に使い、表示には使わない。
	Err error
}

func (e *DownloadNetworkError) Error() string {
	return "署名付きURLへの通信に失敗しました"
}

// Unwrapは、キャンセルやタイムアウトなどの原因を判定できるようにする。
func (e *DownloadNetworkError) Unwrap() error {
	return e.Err
}

// Downloadは、APIが返した署名付きURL (添付ファイルの `download_url`) にGETを送り、成功の応答を返す。
// URL自体が取得の権限を持つため、Authorizationヘッダーのトークンを付けない。
// 接続先と同じく、HTTPSかループバックのHTTPのURLだけに送る。
// 2xx以外の応答は `*DownloadError`、通信の失敗は `*DownloadNetworkError` として返す。
// 成功の応答の本文の読み取りの失敗も `*DownloadNetworkError` にし、保存先への書き込みの失敗と見分けられるようにする。
// エラーの表示には署名付きURLを含めない。
// 成功の応答の本文は呼び出し側が閉じる。
func (c *Client) Download(ctx context.Context, rawURL string) (*http.Response, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" || u.User != nil || !isAllowedURLScheme(u) {
		return nil, ErrInsecureDownloadURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, &DownloadNetworkError{Err: err}
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	// APIへの要求のトランスポートはトークンを付けるため通さず、元のクライアントで送る。
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &DownloadNetworkError{Err: err}
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxErrorBodyBytes))
		_ = resp.Body.Close()
		return nil, &DownloadError{StatusCode: resp.StatusCode}
	}

	resp.Body = &downloadBody{ReadCloser: resp.Body}

	return resp, nil
}

// downloadBodyは、署名付きURLの応答の本文の読み取りの失敗を `*DownloadNetworkError` にする。
type downloadBody struct {
	io.ReadCloser
}

func (b *downloadBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	// io.EOFは読み終えたことを表し、io.Copyなどが値で比べるため包まない。
	if err != nil && !errors.Is(err, io.EOF) {
		err = &DownloadNetworkError{Err: err}
	}

	return n, err
}

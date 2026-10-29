package cimd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	// maxDocumentBytesは読み込む文書の大きさの上限。仕様の推奨の5キロバイトにする
	maxDocumentBytes = 5 * 1024

	// defaultFetchTimeoutは文書の取得のタイムアウト。取得は認可要求の中で行い、利用者を待たせるため短くする
	defaultFetchTimeout = 5 * time.Second

	// minCacheLifetimeとmaxCacheLifetimeは文書をキャッシュする期間の下限と上限。HTTPのキャッシュの
	// ヘッダーに従うが、認可要求のたびに取得しないよう下限を、クライアントが文書を変えたときに
	// 古いリダイレクトURIを使い続けないよう上限を設ける
	minCacheLifetime = 5 * time.Minute
	maxCacheLifetime = 24 * time.Hour
)

// tokenEndpointAuthMethodNoneは、クライアントがトークンエンドポイントで認証しないことを表す認証の方式。
// CIMDのクライアントは事前に秘密を共有できないため、Wikinoはこの方式のクライアントだけを受け付ける
const tokenEndpointAuthMethodNone = "none"

// ErrNotAllowedは、クライアントIDが許可リストのホストのCIMDのクライアントIDでないことを表す
var ErrNotAllowed = errors.New("CIMDのクライアントIDとして受け付けないURL")

// Documentは、取得して検証したCIMDのメタデータの文書
type Document struct {
	ClientID     string
	ClientName   string
	RedirectURIs []string

	// ExpiresAtは文書をキャッシュしてよい期限
	ExpiresAt time.Time
}

// documentJSONは文書のJSONのうち、Wikinoが使う項目
type documentJSON struct {
	ClientID                          string   `json:"client_id"`
	ClientName                        string   `json:"client_name"`
	RedirectURIs                      []string `json:"redirect_uris"`
	TokenEndpointAuthMethod           string   `json:"token_endpoint_auth_method"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
}

// Fetcherは、CIMDのクライアントIDのURLからメタデータの文書を取得する
type Fetcher struct {
	client  *http.Client
	timeout time.Duration
}

// NewFetcherはFetcherを生成する。clientは取得に使うHTTPクライアントで、リダイレクトの扱いは
// Fetcherが上書きする (clientそのものは変えない)
func NewFetcher(client *http.Client) *Fetcher {
	return &Fetcher{client: client, timeout: defaultFetchTimeout}
}

// Fetchは、クライアントIDclientIDのURLから文書を取得して検証し、nowを基準にしたキャッシュの期限を
// 付けて返す。許可リストの外のクライアントIDにはErrNotAllowedを返す。
//
// HTTPのリダイレクトはたどらず、200以外の応答・大きすぎる文書・不正な文書はエラーにする。
// エラーのときは何も返さないため、呼び出し側は失敗をキャッシュしない
func (f *Fetcher) Fetch(ctx context.Context, clientID string, now time.Time) (*Document, error) {
	if !IsAllowed(clientID) {
		return nil, ErrNotAllowed
	}

	ctx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, clientID, nil)
	if err != nil {
		return nil, fmt.Errorf("CIMDの文書の要求の作成に失敗: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	client := *f.client
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("CIMDの文書の取得に失敗: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CIMDの文書の応答のステータスが200でない: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDocumentBytes+1))
	if err != nil {
		return nil, fmt.Errorf("CIMDの文書の読み込みに失敗: %w", err)
	}
	if len(body) > maxDocumentBytes {
		return nil, fmt.Errorf("CIMDの文書が%dバイトを超えている", maxDocumentBytes)
	}

	doc, err := parseDocument(clientID, body)
	if err != nil {
		return nil, err
	}
	doc.ExpiresAt = now.Add(cacheLifetime(resp.Header.Get("Cache-Control")))
	return doc, nil
}

// parseDocumentは文書のJSONを解釈し、取得したURLclientIDの文書として使えるかを確かめる
func parseDocument(clientID string, body []byte) (*Document, error) {
	var raw documentJSON
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("CIMDの文書がJSONとして不正: %w", err)
	}
	if raw.ClientID != clientID {
		return nil, fmt.Errorf("CIMDの文書のclient_idが取得したURLと一致しない: %q", raw.ClientID)
	}
	name := strings.TrimSpace(raw.ClientName)
	if name == "" {
		return nil, errors.New("CIMDの文書にclient_nameが無い")
	}
	if len(raw.RedirectURIs) == 0 || slices.Contains(raw.RedirectURIs, "") {
		return nil, errors.New("CIMDの文書にredirect_urisが無い")
	}
	// 単数形 (RFC 7591) と複数形のどちらかで `none` を宣言していれば受け付ける。両方を載せ、
	// 単数形に別の方式を書く文書 (ChatGPT) もあるため、どちらかに含まれていればよい
	if raw.TokenEndpointAuthMethod != tokenEndpointAuthMethodNone && !slices.Contains(raw.TokenEndpointAuthMethodsSupported, tokenEndpointAuthMethodNone) {
		return nil, errors.New("CIMDの文書の認証の方式にnoneが無い")
	}
	return &Document{
		ClientID:     raw.ClientID,
		ClientName:   name,
		RedirectURIs: raw.RedirectURIs,
	}, nil
}

// cacheLifetimeは、応答のCache-Controlのヘッダーから文書をキャッシュする期間を決める。
// `max-age` に従い、`no-store`・`no-cache` や `max-age` の無い応答は下限の期間にする。
// どの場合も下限と上限の間に収める
func cacheLifetime(cacheControl string) time.Duration {
	lifetime := time.Duration(0)
	for _, directive := range strings.Split(cacheControl, ",") {
		name, value, _ := strings.Cut(strings.TrimSpace(directive), "=")
		switch strings.ToLower(name) {
		case "no-store", "no-cache":
			return minCacheLifetime
		case "max-age":
			// 秒数をDurationに直す前に上限で抑え、大きな値の桁あふれを避ける
			if seconds, err := strconv.ParseInt(strings.Trim(value, `"`), 10, 64); err == nil && seconds > 0 {
				lifetime = time.Duration(min(seconds, int64(maxCacheLifetime/time.Second))) * time.Second
			}
		}
	}
	return min(max(lifetime, minCacheLifetime), maxCacheLifetime)
}

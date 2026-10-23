package apiutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
)

// ErrDryRunは、`--dry-run` で状態を変える要求を送らなかったことを表す。
// コマンドはこのエラーを受け取ったら、DryRunに記録した要求を出して成功させる。
var ErrDryRun = errors.New("apiutil: --dry-runのため要求を送っていません")

// redactedは、伏せた値の代わりに出す文字列。
const redacted = "[REDACTED]"

// DryRunは、`--dry-run` で送らなかった要求を記録する。
// 要求の表示をHTTP層で共通に作り、コマンドごとに組み立てないためのもの。
type DryRun struct {
	// IncludeReadOnlyがtrueなら、読み取り専用の要求も送らずに記録する。
	// 任意の要求を送る `wikino api` で、`--dry-run` なのに要求を送ってしまわないために使う。
	// falseなら状態を変える要求だけを記録し、確認や表示に要る読み取りの要求は送る。
	IncludeReadOnly bool

	mu       sync.Mutex
	requests []Request
}

// DryRunOutputは、`--dry-run` の結果として標準出力に出す内容。
// `dry_run` を必ず含め、実際に送った結果と取り違えないようにする。
type DryRunOutput struct {
	DryRun   bool      `json:"dry_run"`
	Requests []Request `json:"requests"`
}

// Requestは、送らなかった要求の表示。秘密の値は伏せてある。
type Request struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	// BodyはJSONの本文ならその値、それ以外は文字列。本文が無ければ省く。
	Body any `json:"body,omitempty"`
}

// Requestsは、記録した要求を記録した順に返す。
func (d *DryRun) Requests() []Request {
	d.mu.Lock()
	defer d.mu.Unlock()

	return append([]Request(nil), d.requests...)
}

// Outputは、記録した要求を `--dry-run` の結果にする。
func (d *DryRun) Output() DryRunOutput {
	return DryRunOutput{DryRun: true, Requests: d.Requests()}
}

// Doは、すべての要求を送らずに記録する (IncludeReadOnlyのとき、送る代わりに使う)。
func (d *DryRun) Do(req *http.Request) (*http.Response, error) {
	return nil, d.intercept(req)
}

// interceptは、reqを伏せて記録し、送らないようにErrDryRunを返す。
func (d *DryRun) intercept(req *http.Request) error {
	r, err := redactRequest(req)
	if err != nil {
		return err
	}

	d.mu.Lock()
	d.requests = append(d.requests, r)
	d.mu.Unlock()

	return ErrDryRun
}

func redactRequest(req *http.Request) (Request, error) {
	r := Request{
		Method:  req.Method,
		URL:     req.URL.Redacted(),
		Headers: map[string]string{},
	}

	names := make([]string, 0, len(req.Header))
	for name := range req.Header {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		value := strings.Join(req.Header.Values(name), ", ")
		switch http.CanonicalHeaderKey(name) {
		case "Authorization", "Proxy-Authorization":
			value = redactCredentials(value)
		case "Cookie":
			value = redacted
		}
		r.Headers[name] = value
	}

	body, err := readBody(req)
	if err != nil {
		return Request{}, err
	}
	if len(body) > 0 {
		if strings.Contains(req.Header.Get("Content-Type"), "json") && json.Valid(body) {
			r.Body = json.RawMessage(body)
		} else {
			r.Body = string(body)
		}
	}

	return r, nil
}

// redactCredentialsは、認証のヘッダーの値を、スキーム (Bearerなど) だけを残して伏せる。
func redactCredentials(value string) string {
	if scheme, _, ok := strings.Cut(value, " "); ok {
		return scheme + " " + redacted
	}

	return redacted
}

// readBodyは要求の本文を読み、読んだ後も要求から同じ本文を読めるように戻す。
func readBody(req *http.Request) ([]byte, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, nil
	}

	body, err := io.ReadAll(req.Body)
	_ = req.Body.Close()
	if err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewReader(body))

	return body, nil
}

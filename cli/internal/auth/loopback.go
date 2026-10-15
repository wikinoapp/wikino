package auth

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net"
	"net/http"
	"time"
)

// callbackPathは、ループバックで認可の応答を受けるパス。公式クライアントに登録したリダイレクトURIと同じにする。
const callbackPath = "/callback"

// DefaultLoopbackTimeoutは、ブラウザでの許可を待つ時間の上限。
// 応答が無いまま待ち続けて、エージェントが止まり続けないようにする。
const DefaultLoopbackTimeout = 5 * time.Minute

// ErrStateMismatchは、認可の応答のstateが要求と一致しないことを表す。
// 別の要求の応答を受け取らないよう、ログインを中止する。
var ErrStateMismatch = errors.New("認可の応答のstateが要求と一致しません")

// ErrIssuerMismatchは、認可の応答のissが認可サーバーの発行者と一致しないことを表す (RFC 9207)。
var ErrIssuerMismatch = errors.New("認可の応答のissが認可サーバーと一致しません")

// ErrLoopbackTimeoutは、時間内にブラウザから認可の応答が届かなかったことを表す。
var ErrLoopbackTimeout = errors.New("ブラウザからの認可の応答を待つ時間が過ぎました")

// ErrLoopbackUnavailableは、ブラウザを開けないかループバックで待ち受けられず、
// ループバックでは認可の応答を受け取れないことを表す。呼び出し側はデバイス認可に切り替える。
var ErrLoopbackUnavailable = errors.New("ループバックでは認可の応答を受け取れません")

// LoopbackOptionsは、ループバックによるログインの設定。
type LoopbackOptions struct {
	// Spaceはログインするスペースの識別子。
	Space string
	// Scopesは要求するスコープ。
	Scopes []string
	// Openは、認可要求のURLをブラウザで開き、利用者に示す。ブラウザを開けなければエラーを返す。
	Open func(authorizationURL string) error
	// SuccessPageとFailurePageは、認可の応答を受けた後にブラウザへ表示する文言。
	SuccessPage string
	FailurePage string
	// Langはブラウザへ表示する文言の言語 (jaまたはen)。
	Lang string
	// Timeoutはブラウザでの許可を待つ時間の上限。0ならDefaultLoopbackTimeout。
	Timeout time.Duration
}

// LoginWithLoopbackは、ループバック (RFC 8252) で認可コードを受け取り、トークンに交換する。
// 127.0.0.1のOSが選んだポートで待ち受け、PKCE (S256)・state・`resource` を付けて認可を要求し、
// 応答のstateとiss (RFC 9207) を確かめてから交換する。
// 待ち受けやブラウザの起動に失敗したときは、ErrLoopbackUnavailableを包んだエラーを返す。
func (c *OAuthClient) LoginWithLoopback(ctx context.Context, metadata *Metadata, opts LoopbackOptions) (*TokenResponse, error) {
	var lc net.ListenConfig
	listener, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("%w: 待ち受けられません: %w", ErrLoopbackUnavailable, err)
	}
	redirectURI := fmt.Sprintf("http://%s%s", listener.Addr().String(), callbackPath)

	state := NewState()
	pkce := NewPKCE()
	resource := ResourceURL(c.Host, opts.Space)

	results := make(chan callbackResult, 1)
	server := &http.Server{
		Handler:           &callbackHandler{metadata: metadata, state: state, opts: opts, results: results},
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() { _ = server.Serve(listener) }()
	defer func() { _ = server.Close() }()

	if err := opts.Open(metadata.AuthorizationURL(AuthorizationRequest{
		RedirectURI: redirectURI,
		Scopes:      opts.Scopes,
		State:       state,
		PKCE:        pkce,
		Resource:    resource,
	})); err != nil {
		return nil, fmt.Errorf("%w: ブラウザを開けません: %w", ErrLoopbackUnavailable, err)
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = DefaultLoopbackTimeout
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	var result callbackResult
	select {
	case result = <-results:
	case <-timer.C:
		return nil, ErrLoopbackTimeout
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if result.err != nil {
		return nil, result.err
	}

	return c.ExchangeCode(ctx, metadata, result.code, redirectURI, pkce.Verifier, resource)
}

type callbackResult struct {
	code string
	err  error
}

// callbackHandlerは、ブラウザがリダイレクトしてきた認可の応答を受け取る。
// 最初の応答だけを結果にし、以後の応答には失敗のページを返す。
type callbackHandler struct {
	metadata *Metadata
	state    string
	opts     LoopbackOptions
	results  chan callbackResult
}

func (h *callbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// ブラウザがファビコンなどを要求しても、ログインの結果にしない。
	if r.URL.Path != callbackPath || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	result := h.parse(r)
	delivered := false
	select {
	case h.results <- result:
		delivered = true
	default:
	}

	page := h.opts.FailurePage
	if delivered && result.err == nil {
		page = h.opts.SuccessPage
	}
	header := w.Header()
	header.Set("Content-Type", "text/html; charset=utf-8")
	header.Set("Cache-Control", "no-store")
	// アドレスバーのURLには認可コードが入っているため、遷移先に渡さない。
	header.Set("Referrer-Policy", "no-referrer")
	_, _ = fmt.Fprintf(w, "<!doctype html><html lang=\"%s\"><meta charset=\"utf-8\"><title>Wikino CLI</title><body><p>%s</p></body></html>\n", html.EscapeString(h.opts.Lang), html.EscapeString(page))
}

func (h *callbackHandler) parse(r *http.Request) callbackResult {
	query := r.URL.Query()

	// stateが一致しない応答は別の要求のものか偽物のため、エラーの内容も信用しない。
	if query.Get("state") != h.state {
		return callbackResult{err: ErrStateMismatch}
	}
	// 発行者を応答に載せると宣言した認可サーバーなら、issの無い応答も受け付けない。
	if query.Has("iss") || h.metadata.AuthorizationResponseIssParameterSupported {
		if query.Get("iss") != h.metadata.Issuer {
			return callbackResult{err: ErrIssuerMismatch}
		}
	}
	if code := query.Get("error"); code != "" {
		return callbackResult{err: &OAuthError{Code: code}}
	}
	code := query.Get("code")
	if code == "" {
		return callbackResult{err: fmt.Errorf("%w: 認可の応答に認可コードがありません", ErrUnexpectedResponse)}
	}

	return callbackResult{code: code}
}

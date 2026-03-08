package middleware

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/wikinoapp/wikino/go/internal/clientip"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/session"
)

// featureFlagChecker はフィーチャーフラグの有効判定を行うインターフェース
// repository.FeatureFlagRepository がこのインターフェースを満たす
type featureFlagChecker interface {
	IsEnabledBySessionToken(ctx context.Context, sessionToken string, name model.FeatureFlagName) (bool, error)
}

// featureFlaggedPattern はフィーチャーフラグで制御するURLパターンを定義
type featureFlaggedPattern struct {
	pattern *regexp.Regexp
	flag    model.FeatureFlagName
	methods []string // nilまたは空なら全メソッドにマッチ
}

// フィーチャーフラグで制御するURLパターンのリスト
// パターンを追加するには、このスライスに要素を追加する
var featureFlaggedPatterns = []featureFlaggedPattern{}

// ReverseProxyMiddleware はRails版へのリバースプロキシミドルウェア
type ReverseProxyMiddleware struct {
	railsURL        *url.URL
	proxy           *httputil.ReverseProxy
	cfg             *config.Config
	featureFlagRepo featureFlagChecker
}

// Go版で処理するパス（ホワイトリスト）
// これらのパスはRails版にプロキシせず、Go版のハンドラーで処理する
//
// 3種類のマッチング方式：
// - 完全一致（exactPaths）: パスが完全に一致する場合のみマッチ
// - プレフィックス一致（prefixPaths）: パスが指定した文字列で始まる場合にマッチ
// - 正規表現（goHandledRegexPatterns）: 動的セグメントやメソッド制限が必要なパスに使用
var (
	// 完全一致でチェックするパス
	// "/" をプレフィックス一致に追加すると全パスがマッチしてしまうため、完全一致で処理する
	goHandledExactPaths = []string{
		"/",              // トップページ
		"/manifest.json", // Web App Manifest
	}

	// プレフィックス一致でチェックするパス
	goHandledPrefixPaths = []string{
		"/static",                          // 静的ファイル（CSS、JS、画像など）
		"/health",                          // ヘルスチェックエンドポイント
		"/sign_in",                         // ログインページ・処理
		"/sign_up",                         // サインアップページ
		"/email_confirmation",              // メール確認コード送信・検証
		"/accounts",                        // アカウント作成
		"/user_session",                    // セッション作成・削除
		"/sign_in/two_factor/new",          // 2FAコード入力フォーム
		"/sign_in/two_factor/recovery/new", // リカバリーコード入力フォーム
		"/sign_in/two_factor/recovery",     // リカバリーコード検証
		"/sign_in/two_factor",              // 2FAコード検証
		"/password/reset",                  // パスワードリセット申請フォーム・処理
		"/password/edit",                   // 新パスワード入力フォーム
		"/password",                        // パスワード更新処理
		"/drafts",                          // 下書き一覧
	}
)

// goHandledPattern はGo版で常に処理するURLパターンを定義（正規表現 + メソッドフィルタ）
type goHandledPattern struct {
	pattern *regexp.Regexp
	methods []string // nilまたは空なら全メソッドにマッチ
}

// Go版で処理するURLパターン（正規表現マッチング）
// プレフィックス一致では表現できないパス（動的セグメントやメソッド制限が必要なパス）に使用する
var goHandledRegexPatterns = []goHandledPattern{
	{pattern: regexp.MustCompile(`^/s/[^/]+/topics/\d+$`)},
	{pattern: regexp.MustCompile(`^/s/[^/]+/pages/\d+/edit$`)},
	{pattern: regexp.MustCompile(`^/s/[^/]+/pages/\d+/draft_page$`)},
	{pattern: regexp.MustCompile(`^/s/[^/]+/pages/\d+/draft_page_revision$`)},
	{pattern: regexp.MustCompile(`^/s/[^/]+/pages/\d+$`), methods: []string{"PATCH"}},
	{pattern: regexp.MustCompile(`^/s/[^/]+/page_locations$`)},
	{pattern: regexp.MustCompile(`^/s/[^/]+/pages/\d+/link_list$`)},
	{pattern: regexp.MustCompile(`^/s/[^/]+/pages/\d+/links/\d+/backlink_list$`)},
	{pattern: regexp.MustCompile(`^/s/[^/]+/pages/\d+/backlinks$`)},
	{pattern: regexp.MustCompile(`^/s/[^/]+/pages/\d+/move$`)},
}

// NewReverseProxyMiddleware は新しいReverseProxyMiddlewareを作成
// featureFlagRepoがnilの場合、フィーチャーフラグ判定はスキップされる
func NewReverseProxyMiddleware(railsURL string, cfg *config.Config, featureFlagRepo featureFlagChecker) (*ReverseProxyMiddleware, error) {
	parsedURL, err := url.Parse(railsURL)
	if err != nil {
		return nil, err
	}

	// httputil.ReverseProxyを作成
	proxy := httputil.NewSingleHostReverseProxy(parsedURL)

	// カスタムのHTTP Transportを設定（タイムアウトと接続プーリング）
	proxy.Transport = &http.Transport{
		// 接続タイムアウト: 10秒
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		// レスポンスヘッダー読み取りタイムアウト: 30秒
		ResponseHeaderTimeout: 30 * time.Second,
		// 接続プーリングの設定
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// プロキシのディレクターをカスタマイズ（ヘッダー設定）
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		// 既存のX-Forwarded-ForとX-Real-IPを保存
		originalXForwardedFor := req.Header.Get("X-Forwarded-For")
		originalXRealIP := req.Header.Get("X-Real-IP")

		// クライアントIPアドレスを取得
		clientIP := clientip.GetClientIP(req)

		// originalDirectorを呼び出す
		originalDirector(req)

		// X-Forwarded-Forヘッダーの設定
		// 注: httputil.ReverseProxyのServeHTTPメソッドは、Directorを呼び出した後に
		// X-Forwarded-Forヘッダーが存在する場合、RemoteAddrを追加してしまう。
		// これを防ぐために、ヘッダーマップから完全に削除してから再設定する。
		delete(req.Header, "X-Forwarded-For")
		if originalXForwardedFor != "" {
			// 既存の値を維持（Cloudflareなどが設定した値を保持）
			req.Header.Set("X-Forwarded-For", originalXForwardedFor)
		} else {
			// 既存の値がない場合、clientIPを設定
			req.Header.Set("X-Forwarded-For", clientIP)
		}

		// X-Real-IPヘッダーの設定（既存の値がない場合のみ）
		if originalXRealIP != "" {
			req.Header.Set("X-Real-IP", originalXRealIP)
		} else {
			req.Header.Set("X-Real-IP", clientIP)
		}

		// X-Forwarded-Protoの設定
		req.Header.Set("X-Forwarded-Proto", "https")

		// X-Forwarded-Hostの設定
		req.Header.Set("X-Forwarded-Host", cfg.Domain)

		// ログ出力（開発者向け）
		slog.Info("リバースプロキシでRails版にリクエストを転送",
			"path", req.URL.Path,
			"method", req.Method,
			"target", parsedURL.String()+req.URL.Path,
			"client_ip", clientIP,
		)
	}

	// レスポンス処理後のログ出力（成功時）
	proxy.ModifyResponse = func(resp *http.Response) error {
		// プロキシが成功した場合のレスポンスログを出力（開発者向け）
		slog.Info("Rails版からレスポンスを受信",
			"status_code", resp.StatusCode,
			"status", resp.Status,
			"path", resp.Request.URL.Path,
			"method", resp.Request.Method,
		)
		return nil
	}

	// エラーハンドラーをカスタマイズ
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		ctx := r.Context()

		// 詳細なエラーログを出力（開発者向け）
		slog.ErrorContext(ctx, "Rails版へのプロキシでエラーが発生",
			"error", err,
			"path", r.URL.Path,
			"method", r.Method,
			"remote_addr", r.RemoteAddr,
		)

		// 502エラーレスポンスを返す
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		// フォールバックエラーレスポンスなので、書き込みエラーは無視
		_, _ = w.Write([]byte(render502ErrorHTML()))
	}

	return &ReverseProxyMiddleware{
		railsURL:        parsedURL,
		proxy:           proxy,
		cfg:             cfg,
		featureFlagRepo: featureFlagRepo,
	}, nil
}

// Middleware はHTTPミドルウェアを返す
func (m *ReverseProxyMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. 常にGoで処理するパス（完全一致・プレフィックス一致）
		if m.isGoHandledPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// 2. 常にGoで処理するパス（正規表現 + メソッドフィルタ）
		if m.isGoHandledByRegex(r) {
			next.ServeHTTP(w, r)
			return
		}

		// 3. フィーチャーフラグで制御するパス
		if flagName := m.getFeatureFlagForRequest(r); flagName != "" {
			if m.isFeatureFlagEnabled(r, flagName) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// 4. その他はすべてRailsにプロキシ
		m.proxy.ServeHTTP(w, r)
	})
}

// isGoHandledPath はGo版で処理するパスかどうかを判定
func (m *ReverseProxyMiddleware) isGoHandledPath(path string) bool {
	// 完全一致でチェック
	for _, p := range goHandledExactPaths {
		if path == p {
			return true
		}
	}

	// プレフィックス一致でチェック
	for _, p := range goHandledPrefixPaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}

	return false
}

// isGoHandledByRegex はGo版で処理するパスかどうかを正規表現パターンで判定する
// メソッドフィルタが指定されている場合はHTTPメソッドも考慮する
func (m *ReverseProxyMiddleware) isGoHandledByRegex(r *http.Request) bool {
	for _, gp := range goHandledRegexPatterns {
		if !gp.pattern.MatchString(r.URL.Path) {
			continue
		}
		if len(gp.methods) > 0 && !containsMethod(gp.methods, r.Method) {
			continue
		}
		return true
	}
	return false
}

// getFeatureFlagForRequest はHTTPリクエストのパスとメソッドに対応するフィーチャーフラグ名を返す
// マッチするパターンがない場合は空文字列を返す
func (m *ReverseProxyMiddleware) getFeatureFlagForRequest(r *http.Request) model.FeatureFlagName {
	for _, fp := range featureFlaggedPatterns {
		if !fp.pattern.MatchString(r.URL.Path) {
			continue
		}
		if len(fp.methods) > 0 && !containsMethod(fp.methods, r.Method) {
			continue
		}
		return fp.flag
	}
	return ""
}

// containsMethod は指定されたHTTPメソッドがスライスに含まれるかを判定する
//
// HTMLフォームはGETとPOSTのみサポートするため、PATCH/PUT/DELETEリクエストは
// POST + _methodパラメータとして送信される（Method Overrideパターン）。
// このミドルウェアはMethod Overrideミドルウェアより前に実行されるため、
// POSTリクエストもPATCH/PUT/DELETEパターンにマッチさせる必要がある。
func containsMethod(methods []string, method string) bool {
	for _, m := range methods {
		if m == method {
			return true
		}
	}

	// POSTリクエストはMethod Override経由でPATCH/PUT/DELETEに変換される可能性がある
	if method == http.MethodPost {
		for _, m := range methods {
			switch m {
			case http.MethodPatch, http.MethodPut, http.MethodDelete:
				return true
			}
		}
	}

	return false
}

// isFeatureFlagEnabled はリクエストのセッションCookieからユーザーを特定し、
// フィーチャーフラグが有効かどうかを判定する
// エラー時またはCookieなし時はfalseを返す（Rails版にフォールバック）
func (m *ReverseProxyMiddleware) isFeatureFlagEnabled(r *http.Request, flagName model.FeatureFlagName) bool {
	if m.featureFlagRepo == nil {
		return false
	}

	cookie, err := r.Cookie(session.CookieName)
	if err != nil || cookie.Value == "" {
		return false
	}

	enabled, err := m.featureFlagRepo.IsEnabledBySessionToken(r.Context(), cookie.Value, flagName)
	if err != nil {
		slog.WarnContext(r.Context(), "フィーチャーフラグ判定でエラーが発生（Rails版にフォールバック）",
			"error", err,
			"flag", flagName,
			"path", r.URL.Path,
		)
		return false
	}

	return enabled
}

// render502ErrorHTML は502エラーページのHTMLを返す
func render502ErrorHTML() string {
	return `<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>サービス接続エラー - Wikino</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            margin: 0;
            padding: 0;
            display: flex;
            align-items: center;
            justify-content: center;
            min-height: 100vh;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 600px;
            padding: 2rem;
            text-align: center;
        }
        h1 {
            font-size: 2rem;
            color: #333;
            margin-bottom: 1rem;
        }
        p {
            color: #666;
            line-height: 1.6;
            margin-bottom: 2rem;
        }
        a {
            display: inline-block;
            padding: 0.75rem 1.5rem;
            background-color: #3b82f6;
            color: white;
            text-decoration: none;
            border-radius: 0.375rem;
            transition: background-color 0.2s;
        }
        a:hover {
            background-color: #2563eb;
        }
        .icon {
            font-size: 4rem;
            margin-bottom: 1rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">⚠️</div>
        <h1>サービス接続エラー</h1>
        <p>申し訳ございません。現在サービスに接続できません。<br>しばらくしてから再度お試しください</p>
        <a href="/">トップページに戻る</a>
    </div>
</body>
</html>`
}

// Package config はアプリケーション設定の管理機能を提供します
package config

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Config はアプリケーションの設定を保持する構造体です
type Config struct {
	// 環境
	Env string

	// データベース
	DatabaseURL string

	// サーバー
	Port   string
	Domain string

	// Cookie設定
	CookieDomain string

	// セッション
	SessionSecure   bool
	SessionHTTPOnly bool

	// Rate Limiting設定
	DisableRateLimit bool

	// Rails版アプリのURL（リバースプロキシ用）
	RailsAppURL string

	// Cloudflare Turnstile（Bot対策）
	TurnstileEnabled   bool
	TurnstileSiteKey   string
	TurnstileSecretKey string

	// メンテナンスモード
	MaintenanceMode bool
	AdminIPs        []string

	// アセットバージョン（CDNキャッシュ対策用）
	AssetVersion string

	// Resend（メール送信）
	ResendAPIKey    string
	ResendFromEmail string
	ResendFromName  string

	// imgproxy（画像配信） - og:image エンドポイントのリサイズ・フォーマット変換に使用
	// ImgproxyURL は signed URL のベース URL（ブラウザがアクセスする URL）
	// ImgproxyKey / ImgproxySalt は HMAC-SHA256 署名用の 16 進数文字列
	ImgproxyURL  string
	ImgproxyKey  string
	ImgproxySalt string

	// 添付ファイルの保存先S3互換ストレージ
	// R2BucketName は imgproxy に渡す元画像 URL "s3://{bucket}/{key}" の構築に使う
	R2BucketName string

	// Sentry (error tracking)
	// [Ja] Sentry (エラー追跡)
	SentryDSN              string
	SentryEnvironment      string
	SentryTracesSampleRate float64
	SentryDebug            bool
}

// Load は環境変数から設定を読み込みます
func Load() (*Config, error) {
	// APP_ENVの値を取得（デフォルト: dev）
	// dev: 開発環境、test: テスト環境、prod: 本番環境
	//
	// すべての環境でGoプロセス起動時には既に環境変数がセット済みです：
	// - ローカル開発/テスト: op run --env-file=".env" が処理済み
	// - CI環境: GitHub Actionsが設定済み
	// - 本番環境: Dokkuが設定済み
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	cfg := &Config{
		Env: env,
	}

	// 必須の環境変数をチェック
	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("必須の環境変数 DATABASE_URL が設定されていません")
	}

	cfg.Port = os.Getenv("WIKINO_PORT")
	if cfg.Port == "" {
		return nil, fmt.Errorf("必須の環境変数 WIKINO_PORT が設定されていません")
	}

	cfg.Domain = os.Getenv("WIKINO_DOMAIN")
	if cfg.Domain == "" {
		return nil, fmt.Errorf("必須の環境変数 WIKINO_DOMAIN が設定されていません")
	}

	cfg.CookieDomain = os.Getenv("WIKINO_COOKIE_DOMAIN")
	if cfg.CookieDomain == "" {
		return nil, fmt.Errorf("必須の環境変数 WIKINO_COOKIE_DOMAIN が設定されていません")
	}

	sessionSecureStr := os.Getenv("WIKINO_SESSION_SECURE")
	if sessionSecureStr == "" {
		return nil, fmt.Errorf("必須の環境変数 WIKINO_SESSION_SECURE が設定されていません")
	}
	cfg.SessionSecure = sessionSecureStr == "true"

	sessionHTTPOnlyStr := os.Getenv("WIKINO_SESSION_HTTPONLY")
	if sessionHTTPOnlyStr == "" {
		return nil, fmt.Errorf("必須の環境変数 WIKINO_SESSION_HTTPONLY が設定されていません")
	}
	cfg.SessionHTTPOnly = sessionHTTPOnlyStr == "true"

	// Rate Limiting設定（オプショナル - 開発環境でRate Limitingを無効化）
	cfg.DisableRateLimit = os.Getenv("WIKINO_DISABLE_RATE_LIMIT") == "true"

	// Rails版アプリのURL（オプショナル - リバースプロキシ機能で使用）
	cfg.RailsAppURL = os.Getenv("WIKINO_RAILS_APP_URL")

	// Cloudflare Turnstile（Bot対策 - ログイン・サインアップフォームで使用）
	// WIKINO_TURNSTILE_ENABLED が "false" の場合はTurnstile検証を無効化する
	// 未設定またはそれ以外の値の場合は有効（デフォルト: 有効）
	cfg.TurnstileEnabled = os.Getenv("WIKINO_TURNSTILE_ENABLED") != "false"
	cfg.TurnstileSiteKey = os.Getenv("WIKINO_TURNSTILE_SITE_KEY")
	cfg.TurnstileSecretKey = os.Getenv("WIKINO_TURNSTILE_SECRET_KEY")

	// メンテナンスモード（オプショナル - "on"のときメンテナンスモードを有効化）
	cfg.MaintenanceMode = os.Getenv("WIKINO_MAINTENANCE_MODE") == "on"

	// 管理者IP（オプショナル - カンマ区切りで複数指定可能）
	adminIPStr := os.Getenv("WIKINO_ADMIN_IP")
	if adminIPStr != "" {
		cfg.AdminIPs = parseAdminIPs(adminIPStr)
	}

	// アセットバージョン（Gitコミットハッシュ）を設定
	cfg.AssetVersion = getGitCommitHash()

	// Resend（メール送信）設定（オプショナル - テスト環境ではモックを使用）
	cfg.ResendAPIKey = os.Getenv("WIKINO_RESEND_API_KEY")
	cfg.ResendFromEmail = os.Getenv("WIKINO_RESEND_FROM_EMAIL")
	cfg.ResendFromName = os.Getenv("WIKINO_RESEND_FROM_NAME")

	// imgproxy（オプショナル - フィーチャーフラグ有効時のみ使用）
	cfg.ImgproxyURL = os.Getenv("WIKINO_IMGPROXY_URL")
	cfg.ImgproxyKey = os.Getenv("WIKINO_IMGPROXY_KEY")
	cfg.ImgproxySalt = os.Getenv("WIKINO_IMGPROXY_SALT")

	// S3互換ストレージのバケット名（imgproxy のソース URL 構築に使用）
	cfg.R2BucketName = os.Getenv("WIKINO_R2_BUCKET_NAME")

	// Sentry (optional — error tracking service).
	// An empty DSN disables Sentry entirely.
	//
	// [Ja] Sentry (オプショナル - エラー追跡サービス)。
	// DSN が空のときは Sentry を完全に無効化する。
	cfg.SentryDSN = os.Getenv("WIKINO_SENTRY_DSN")
	cfg.SentryEnvironment = os.Getenv("WIKINO_SENTRY_ENVIRONMENT")
	if cfg.SentryEnvironment == "" {
		cfg.SentryEnvironment = env
	}
	cfg.SentryTracesSampleRate = parseSentryTracesSampleRate(os.Getenv("WIKINO_SENTRY_TRACES_SAMPLE_RATE"))
	cfg.SentryDebug = os.Getenv("WIKINO_SENTRY_DEBUG") == "true"

	return cfg, nil
}

// DatabaseDSN は PostgreSQL 接続文字列を返します
func (c *Config) DatabaseDSN() string {
	return c.DatabaseURL
}

// IsDev は開発環境かどうかを返します
func (c *Config) IsDev() bool {
	return c.Env == "dev"
}

// IsTest はテスト環境かどうかを返します
func (c *Config) IsTest() bool {
	return c.Env == "test"
}

// IsProduction は本番環境かどうかを返します
func (c *Config) IsProduction() bool {
	return c.Env == "prod"
}

// AppURL はアプリケーションのベースURLを返します
func (c *Config) AppURL() string {
	return "https://" + c.Domain
}

// getGitCommitHash returns the short Git commit hash of the running build. It
// is used as the Sentry release and as the CSS/JS query parameter for CDN cache
// busting.
//
// GIT_REV takes precedence: on Dokku the deployed container has no .git
// directory, so `git rev-parse` fails there and the value would fall back to
// "dev". Dokku instead exposes the deploy commit hash via the GIT_REV
// environment variable, which is provided by the platform (so it carries no
// WIKINO_ prefix). The local git command is the development fallback, and "dev"
// is the last resort.
//
// [Ja] 実行中ビルドの Git コミットハッシュ (短縮版) を返す。Sentry の release と、
// CDN キャッシュ対策用の CSS/JS クエリパラメータに使う。
//
// GIT_REV を最優先する。Dokku のデプロイ先コンテナには .git ディレクトリが無いため
// `git rev-parse` は失敗し、そのままだと "dev" にフォールバックしてしまう。Dokku は
// 代わりにデプロイ時のコミットハッシュを GIT_REV 環境変数で渡す (プラットフォームが
// 提供する変数なので WIKINO_ プレフィックスは付けない)。ローカルの git コマンドは
// 開発用のフォールバックで、最後の手段が "dev"。
func getGitCommitHash() string {
	// Dokku provides the full deploy SHA here; shorten it to the 7-character
	// form that the local `git rev-parse --short` path also produces.
	//
	// [Ja] Dokku はここに完全なデプロイ SHA を渡すので、ローカルの
	// `git rev-parse --short` と同じ 7 文字の短縮形に揃える。
	if rev := strings.TrimSpace(os.Getenv("GIT_REV")); rev != "" {
		const shortHashLen = 7
		if len(rev) > shortHashLen {
			return rev[:shortHashLen]
		}
		return rev
	}

	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		// Fall back to "dev" when git is unavailable (development environment).
		//
		// [Ja] Git が利用できない場合は "dev" を返す (開発環境用のフォールバック)。
		return "dev"
	}
	return strings.TrimSpace(string(out))
}

// GetAssetVersion はアセットのバージョン文字列を返します
// 開発環境: 現在時刻のUnixタイムスタンプ（ミリ秒）を返す（キャッシュを無効化）
// 本番/テスト環境: Gitコミットハッシュを返す（起動時に設定された値）
func (c *Config) GetAssetVersion() string {
	if c.IsDev() {
		// 開発環境では毎回異なる値を返す（現在時刻のUnixタイムスタンプ、ミリ秒）
		return strconv.FormatInt(time.Now().UnixMilli(), 10)
	}
	// 本番/テスト環境では起動時に設定されたGitコミットハッシュを返す
	return c.AssetVersion
}

// parseAdminIPs はカンマ区切りのIP文字列をスライスに変換します
// 各IPアドレスの前後の空白は除去されます
func parseAdminIPs(s string) []string {
	parts := strings.Split(s, ",")
	ips := make([]string, 0, len(parts))
	for _, p := range parts {
		ip := strings.TrimSpace(p)
		if ip != "" {
			ips = append(ips, ip)
		}
	}
	return ips
}

// parseSentryTracesSampleRate parses the Sentry traces sample rate from a string.
// Returns the default 0.5 for empty input, parse failures, and out-of-range values
// (less than 0.0 or greater than 1.0).
//
// [Ja] 文字列から Sentry トレースサンプリングレートをパースする。
// 空文字列、パース失敗、範囲外 (0.0 未満または 1.0 超過) の場合はデフォルト値 0.5 を返す。
func parseSentryTracesSampleRate(s string) float64 {
	if s == "" {
		return 0.5
	}
	rate, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.5
	}
	if rate < 0.0 || rate > 1.0 {
		return 0.5
	}
	return rate
}

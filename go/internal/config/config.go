// Package configはアプリケーション設定の管理機能を提供します
package config

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Configはアプリケーションの設定を保持する構造体です
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

	// Rails版アプリのURL (リバースプロキシ用)
	RailsAppURL string

	// Cloudflare Turnstile (Bot対策)
	TurnstileEnabled   bool
	TurnstileSiteKey   string
	TurnstileSecretKey string

	// メンテナンスモード
	MaintenanceMode bool
	AdminIPs        []string

	// GitRevはデプロイのGitコミットハッシュ (短縮版)。Sentryのreleaseに
	// そのまま使うほか、GetAssetVersion経由で本番/テスト環境のCDNキャッシュ対策用
	// アセットバージョンとしても使う。
	GitRev string

	// Resend (メール送信)
	ResendAPIKey    string
	ResendFromEmail string
	ResendFromName  string

	// imgproxy (画像配信) - og:imageエンドポイントのリサイズ・フォーマット変換に使用
	// ImgproxyURLはsigned URLのベースURL (ブラウザがアクセスするURL)
	// ImgproxyKey / ImgproxySaltはHMAC-SHA256署名用の16進数文字列
	ImgproxyURL  string
	ImgproxyKey  string
	ImgproxySalt string

	// 添付ファイルと、エクスポートが書き出すアーカイブを保持するS3互換オブジェクトストレージ
	// (Cloudflare R2)。R2BucketNameはimgproxyに渡す元画像URL "s3://{bucket}/{key}" の構築にも
	// 使う。エンドポイント・資格情報・リージョンは、internal/storageがバケット自体へ到達するために
	// 必要な設定である。
	R2BucketName      string
	R2Endpoint        string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2Region          string

	// Sentry (エラー追跡)
	SentryDSN              string
	SentryEnvironment      string
	SentryTracesSampleRate float64
	SentryDebug            bool
}

// Loadは環境変数から設定を読み込みます
func Load() (*Config, error) {
	// APP_ENVの値を取得 (デフォルト: dev)
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
		return nil, fmt.Errorf("必須の環境変数DATABASE_URLが設定されていません")
	}

	cfg.Port = os.Getenv("WIKINO_PORT")
	if cfg.Port == "" {
		return nil, fmt.Errorf("必須の環境変数WIKINO_PORTが設定されていません")
	}

	cfg.Domain = os.Getenv("WIKINO_DOMAIN")
	if cfg.Domain == "" {
		return nil, fmt.Errorf("必須の環境変数WIKINO_DOMAINが設定されていません")
	}

	cfg.CookieDomain = os.Getenv("WIKINO_COOKIE_DOMAIN")
	if cfg.CookieDomain == "" {
		return nil, fmt.Errorf("必須の環境変数WIKINO_COOKIE_DOMAINが設定されていません")
	}

	sessionSecureStr := os.Getenv("WIKINO_SESSION_SECURE")
	if sessionSecureStr == "" {
		return nil, fmt.Errorf("必須の環境変数WIKINO_SESSION_SECUREが設定されていません")
	}
	cfg.SessionSecure = sessionSecureStr == "true"

	sessionHTTPOnlyStr := os.Getenv("WIKINO_SESSION_HTTPONLY")
	if sessionHTTPOnlyStr == "" {
		return nil, fmt.Errorf("必須の環境変数WIKINO_SESSION_HTTPONLYが設定されていません")
	}
	cfg.SessionHTTPOnly = sessionHTTPOnlyStr == "true"

	// Rate Limiting設定 (オプショナル - 開発環境でRate Limitingを無効化)
	cfg.DisableRateLimit = os.Getenv("WIKINO_DISABLE_RATE_LIMIT") == "true"

	// Rails版アプリのURL (オプショナル - リバースプロキシ機能で使用)
	cfg.RailsAppURL = os.Getenv("WIKINO_RAILS_APP_URL")

	// Cloudflare Turnstile (Bot対策 - ログイン・サインアップフォームで使用)。
	// WIKINO_TURNSTILE_ENABLEDはstrconv.ParseBoolで解釈するため、"false" / "0" /
	// "FALSE" はいずれもTurnstile検証を無効化する。未設定 (または空文字列) の場合は
	// 有効 (デフォルト: 有効)。真偽値として解釈できない値は、黙って有効側に倒さず
	// エラーで起動を止める。
	//
	// ただし本番環境では、無効化が指定されても無視して有効を維持する。
	// 誤設定でBot対策が黙って無効になることを防ぐため (fail-closed)。非本番環境では
	// 従来どおり無効化を反映する。
	cfg.TurnstileEnabled = true
	if turnstileEnabledStr := os.Getenv("WIKINO_TURNSTILE_ENABLED"); turnstileEnabledStr != "" {
		turnstileEnabled, err := strconv.ParseBool(turnstileEnabledStr)
		if err != nil {
			return nil, fmt.Errorf("環境変数WIKINO_TURNSTILE_ENABLEDを真偽値として解釈できません: %q", turnstileEnabledStr)
		}
		cfg.TurnstileEnabled = turnstileEnabled
	}
	if !cfg.TurnstileEnabled && cfg.IsProduction() {
		// 本番では無効化を無視してTurnstileを有効のままにし、設定を上書きした
		// ことを記録する (開発者向けの運用ログ)。
		slog.Warn("本番環境ではWIKINO_TURNSTILE_ENABLEDによる無効化を無視し、Turnstileを有効のまま維持します (fail-closed)")
		cfg.TurnstileEnabled = true
	}
	cfg.TurnstileSiteKey = os.Getenv("WIKINO_TURNSTILE_SITE_KEY")
	cfg.TurnstileSecretKey = os.Getenv("WIKINO_TURNSTILE_SECRET_KEY")

	// メンテナンスモード (オプショナル - "on"のときメンテナンスモードを有効化)
	cfg.MaintenanceMode = os.Getenv("WIKINO_MAINTENANCE_MODE") == "on"

	// 管理者IP (オプショナル - カンマ区切りで複数指定可能)
	adminIPStr := os.Getenv("WIKINO_ADMIN_IP")
	if adminIPStr != "" {
		cfg.AdminIPs = parseAdminIPs(adminIPStr)
	}

	// 起動時に取得したGitコミットハッシュを設定する。
	cfg.GitRev = getGitCommitHash()

	// Resend (メール送信) 設定 (オプショナル - テスト環境ではモックを使用)
	cfg.ResendAPIKey = os.Getenv("WIKINO_RESEND_API_KEY")
	cfg.ResendFromEmail = os.Getenv("WIKINO_RESEND_FROM_EMAIL")
	cfg.ResendFromName = os.Getenv("WIKINO_RESEND_FROM_NAME")

	// imgproxy (オプショナル - フィーチャーフラグ有効時のみ使用)
	cfg.ImgproxyURL = os.Getenv("WIKINO_IMGPROXY_URL")
	cfg.ImgproxyKey = os.Getenv("WIKINO_IMGPROXY_KEY")
	cfg.ImgproxySalt = os.Getenv("WIKINO_IMGPROXY_SALT")

	// S3互換オブジェクトストレージ (オプショナル)。値はそのまま読み、足りないものは必要と
	// するパッケージが構築時に報告する。ここで必須にしないことで、imgproxyもエクスポートも使わない
	// デプロイがどれも設定せずに起動できる状態を保つ。
	cfg.R2BucketName = os.Getenv("WIKINO_R2_BUCKET_NAME")
	cfg.R2Endpoint = os.Getenv("WIKINO_R2_ENDPOINT")
	cfg.R2AccessKeyID = os.Getenv("WIKINO_R2_ACCESS_KEY_ID")
	cfg.R2SecretAccessKey = os.Getenv("WIKINO_R2_SECRET_ACCESS_KEY")
	cfg.R2Region = os.Getenv("WIKINO_R2_REGION")

	// Sentry (オプショナル - エラー追跡サービス)。
	// DSNが空のときはSentryを完全に無効化する。
	cfg.SentryDSN = os.Getenv("WIKINO_SENTRY_DSN")
	cfg.SentryEnvironment = os.Getenv("WIKINO_SENTRY_ENVIRONMENT")
	if cfg.SentryEnvironment == "" {
		cfg.SentryEnvironment = env
	}
	cfg.SentryTracesSampleRate = parseSentryTracesSampleRate(os.Getenv("WIKINO_SENTRY_TRACES_SAMPLE_RATE"))
	cfg.SentryDebug = os.Getenv("WIKINO_SENTRY_DEBUG") == "true"

	return cfg, nil
}

// DatabaseDSNはPostgreSQL接続文字列を返します
func (c *Config) DatabaseDSN() string {
	return c.DatabaseURL
}

// IsDevは開発環境かどうかを返します
func (c *Config) IsDev() bool {
	return c.Env == "dev"
}

// IsTestはテスト環境かどうかを返します
func (c *Config) IsTest() bool {
	return c.Env == "test"
}

// IsProductionは本番環境かどうかを返します
func (c *Config) IsProduction() bool {
	return c.Env == "prod"
}

// AppURLはアプリケーションのベースURLを返します
func (c *Config) AppURL() string {
	return "https://" + c.Domain
}

// 実行中ビルドのGitコミットハッシュ (短縮版) を返す。Sentryのreleaseと、
// CDNキャッシュ対策用のCSS/JSクエリパラメータに使う。
//
// GIT_REVを最優先する。Dokkuのデプロイ先コンテナには .gitディレクトリが無いため
// `git rev-parse` は失敗し、そのままだと "dev" にフォールバックしてしまう。Dokkuは
// 代わりにデプロイ時のコミットハッシュをGIT_REV環境変数で渡す (プラットフォームが
// 提供する変数なのでWIKINO_ プレフィックスは付けない)。ローカルのgitコマンドは
// 開発用のフォールバックで、最後の手段が "dev"。
func getGitCommitHash() string {
	// Dokkuはここに完全なデプロイSHAを渡すので、7文字に短縮して
	// ローカルの `git rev-parse --short` が返す短縮形におおよそ揃える。
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
		// Gitが利用できない場合は "dev" を返す (開発環境用のフォールバック)。
		return "dev"
	}
	return strings.TrimSpace(string(out))
}

// GetAssetVersionはアセットのバージョン文字列を返します
// 開発環境: 現在時刻のUnixタイムスタンプ (ミリ秒) を返す (キャッシュを無効化)
// 本番/テスト環境: Gitコミットハッシュを返す (起動時に設定された値)
func (c *Config) GetAssetVersion() string {
	if c.IsDev() {
		// 開発環境では毎回異なる値を返す (現在時刻のUnixタイムスタンプ、ミリ秒)
		return strconv.FormatInt(time.Now().UnixMilli(), 10)
	}
	// 本番/テスト環境では起動時に設定されたGitコミットハッシュを返す
	return c.GitRev
}

// parseAdminIPsはカンマ区切りのIP文字列をスライスに変換します
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

// 文字列からSentryトレースサンプリングレートをパースする。
// 空文字列、パース失敗、範囲外 (0.0未満または1.0超過) の場合はデフォルト値0.5を返す。
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

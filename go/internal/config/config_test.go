package config

import (
	"os"
	"reflect"
	"testing"
)

// setupTestEnv は必須の環境変数を設定するヘルパー関数です
func setupTestEnv(t *testing.T) func() {
	t.Helper()

	// 既存の環境変数を保存
	savedEnvs := map[string]string{
		"APP_ENV":                          os.Getenv("APP_ENV"),
		"DATABASE_URL":                     os.Getenv("DATABASE_URL"),
		"WIKINO_PORT":                      os.Getenv("WIKINO_PORT"),
		"WIKINO_DOMAIN":                    os.Getenv("WIKINO_DOMAIN"),
		"WIKINO_COOKIE_DOMAIN":             os.Getenv("WIKINO_COOKIE_DOMAIN"),
		"WIKINO_SESSION_SECURE":            os.Getenv("WIKINO_SESSION_SECURE"),
		"WIKINO_SESSION_HTTPONLY":          os.Getenv("WIKINO_SESSION_HTTPONLY"),
		"WIKINO_DISABLE_RATE_LIMIT":        os.Getenv("WIKINO_DISABLE_RATE_LIMIT"),
		"WIKINO_RAILS_APP_URL":             os.Getenv("WIKINO_RAILS_APP_URL"),
		"WIKINO_TURNSTILE_SITE_KEY":        os.Getenv("WIKINO_TURNSTILE_SITE_KEY"),
		"WIKINO_TURNSTILE_SECRET_KEY":      os.Getenv("WIKINO_TURNSTILE_SECRET_KEY"),
		"WIKINO_MAINTENANCE_MODE":          os.Getenv("WIKINO_MAINTENANCE_MODE"),
		"WIKINO_ADMIN_IP":                  os.Getenv("WIKINO_ADMIN_IP"),
		"WIKINO_SENTRY_DSN":                os.Getenv("WIKINO_SENTRY_DSN"),
		"WIKINO_SENTRY_ENVIRONMENT":        os.Getenv("WIKINO_SENTRY_ENVIRONMENT"),
		"WIKINO_SENTRY_TRACES_SAMPLE_RATE": os.Getenv("WIKINO_SENTRY_TRACES_SAMPLE_RATE"),
		"WIKINO_SENTRY_DEBUG":              os.Getenv("WIKINO_SENTRY_DEBUG"),
	}

	// 必須の環境変数を設定
	_ = os.Setenv("APP_ENV", "test")
	_ = os.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/wikino_test")
	_ = os.Setenv("WIKINO_PORT", "8080")
	_ = os.Setenv("WIKINO_DOMAIN", "test.wikino.app")
	_ = os.Setenv("WIKINO_COOKIE_DOMAIN", ".test.wikino.app")
	_ = os.Setenv("WIKINO_SESSION_SECURE", "false")
	_ = os.Setenv("WIKINO_SESSION_HTTPONLY", "true")

	// Reset Sentry env vars so each test can control them independently.
	// [Ja] Sentry 関連は各テストが独立して状態を制御できるよう、デフォルトでは未設定にする。
	_ = os.Unsetenv("WIKINO_SENTRY_DSN")
	_ = os.Unsetenv("WIKINO_SENTRY_ENVIRONMENT")
	_ = os.Unsetenv("WIKINO_SENTRY_TRACES_SAMPLE_RATE")
	_ = os.Unsetenv("WIKINO_SENTRY_DEBUG")

	// クリーンアップ関数を返す
	return func() {
		for key, value := range savedEnvs {
			if value != "" {
				_ = os.Setenv(key, value)
			} else {
				_ = os.Unsetenv(key)
			}
		}
	}
}

// TestLoad は環境変数から設定を読み込むテスト
func TestLoad(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// 基本的な設定が読み込まれていることを確認
	if cfg.DatabaseURL == "" {
		t.Error("DatabaseURL should not be empty")
	}
	if cfg.Port == "" {
		t.Error("Port should not be empty")
	}
	if cfg.Env != "test" {
		t.Errorf("Env = %v, want test", cfg.Env)
	}
	if cfg.Domain != "test.wikino.app" {
		t.Errorf("Domain = %v, want test.wikino.app", cfg.Domain)
	}
	if cfg.CookieDomain != ".test.wikino.app" {
		t.Errorf("CookieDomain = %v, want .test.wikino.app", cfg.CookieDomain)
	}
	if cfg.SessionSecure != false {
		t.Errorf("SessionSecure = %v, want false", cfg.SessionSecure)
	}
	if cfg.SessionHTTPOnly != true {
		t.Errorf("SessionHTTPOnly = %v, want true", cfg.SessionHTTPOnly)
	}
}

// TestLoad_MissingDatabaseURL は DATABASE_URL が未設定の場合のエラーをテスト
func TestLoad_MissingDatabaseURL(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	_ = os.Unsetenv("DATABASE_URL")

	_, err := Load()
	if err == nil {
		t.Error("Load() should return error when DATABASE_URL is missing")
	}
}

// TestLoad_MissingPort は WIKINO_PORT が未設定の場合のエラーをテスト
func TestLoad_MissingPort(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	_ = os.Unsetenv("WIKINO_PORT")

	_, err := Load()
	if err == nil {
		t.Error("Load() should return error when WIKINO_PORT is missing")
	}
}

// TestLoad_MissingDomain は WIKINO_DOMAIN が未設定の場合のエラーをテスト
func TestLoad_MissingDomain(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	_ = os.Unsetenv("WIKINO_DOMAIN")

	_, err := Load()
	if err == nil {
		t.Error("Load() should return error when WIKINO_DOMAIN is missing")
	}
}

// TestLoad_MissingCookieDomain は WIKINO_COOKIE_DOMAIN が未設定の場合のエラーをテスト
func TestLoad_MissingCookieDomain(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	_ = os.Unsetenv("WIKINO_COOKIE_DOMAIN")

	_, err := Load()
	if err == nil {
		t.Error("Load() should return error when WIKINO_COOKIE_DOMAIN is missing")
	}
}

// TestLoad_MissingSessionSecure は WIKINO_SESSION_SECURE が未設定の場合のエラーをテスト
func TestLoad_MissingSessionSecure(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	_ = os.Unsetenv("WIKINO_SESSION_SECURE")

	_, err := Load()
	if err == nil {
		t.Error("Load() should return error when WIKINO_SESSION_SECURE is missing")
	}
}

// TestLoad_MissingSessionHTTPOnly は WIKINO_SESSION_HTTPONLY が未設定の場合のエラーをテスト
func TestLoad_MissingSessionHTTPOnly(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	_ = os.Unsetenv("WIKINO_SESSION_HTTPONLY")

	_, err := Load()
	if err == nil {
		t.Error("Load() should return error when WIKINO_SESSION_HTTPONLY is missing")
	}
}

// TestLoad_DefaultEnv は APP_ENV が未設定の場合のデフォルト値をテスト
func TestLoad_DefaultEnv(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	_ = os.Unsetenv("APP_ENV")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Env != "dev" {
		t.Errorf("Env = %v, want dev (default)", cfg.Env)
	}
}

// TestDatabaseDSN は DatabaseDSN メソッドをテスト
func TestDatabaseDSN(t *testing.T) {
	cfg := &Config{
		DatabaseURL: "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
	}

	dsn := cfg.DatabaseDSN()
	expected := "postgres://user:pass@localhost:5432/testdb?sslmode=disable"

	if dsn != expected {
		t.Errorf("DatabaseDSN() = %v, want %v", dsn, expected)
	}
}

// TestIsDev は IsDev メソッドをテスト
func TestIsDev(t *testing.T) {
	tests := []struct {
		env  string
		want bool
	}{
		{"dev", true},
		{"test", false},
		{"prod", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cfg := &Config{Env: tt.env}
			if got := cfg.IsDev(); got != tt.want {
				t.Errorf("IsDev() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsTest は IsTest メソッドをテスト
func TestIsTest(t *testing.T) {
	tests := []struct {
		env  string
		want bool
	}{
		{"dev", false},
		{"test", true},
		{"prod", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cfg := &Config{Env: tt.env}
			if got := cfg.IsTest(); got != tt.want {
				t.Errorf("IsTest() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsProduction は IsProduction メソッドをテスト
func TestIsProduction(t *testing.T) {
	tests := []struct {
		env  string
		want bool
	}{
		{"dev", false},
		{"test", false},
		{"prod", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cfg := &Config{Env: tt.env}
			if got := cfg.IsProduction(); got != tt.want {
				t.Errorf("IsProduction() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAppURL は AppURL メソッドをテスト
func TestAppURL(t *testing.T) {
	tests := []struct {
		env    string
		domain string
		want   string
	}{
		{"dev", "localhost", "https://localhost"},
		{"test", "test.wikino.app", "https://test.wikino.app"},
		{"prod", "wikino.app", "https://wikino.app"},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cfg := &Config{Env: tt.env, Domain: tt.domain}
			if got := cfg.AppURL(); got != tt.want {
				t.Errorf("AppURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestLoad_SessionSecure は WIKINO_SESSION_SECURE の bool 変換をテスト
func TestLoad_SessionSecure(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"true", "true", true},
		{"false", "false", false},
		{"other", "yes", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnv(t)
			defer cleanup()

			_ = os.Setenv("WIKINO_SESSION_SECURE", tt.value)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() failed: %v", err)
			}

			if cfg.SessionSecure != tt.want {
				t.Errorf("SessionSecure = %v, want %v", cfg.SessionSecure, tt.want)
			}
		})
	}
}

// TestLoad_SessionHTTPOnly は WIKINO_SESSION_HTTPONLY の bool 変換をテスト
func TestLoad_SessionHTTPOnly(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"true", "true", true},
		{"false", "false", false},
		{"other", "yes", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnv(t)
			defer cleanup()

			_ = os.Setenv("WIKINO_SESSION_HTTPONLY", tt.value)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() failed: %v", err)
			}

			if cfg.SessionHTTPOnly != tt.want {
				t.Errorf("SessionHTTPOnly = %v, want %v", cfg.SessionHTTPOnly, tt.want)
			}
		})
	}
}

// TestLoad_TurnstileConfig は Turnstile 環境変数の読み込みをテスト
func TestLoad_TurnstileConfig(t *testing.T) {
	tests := []struct {
		name          string
		siteKey       string
		secretKey     string
		wantSiteKey   string
		wantSecretKey string
	}{
		{
			name:          "両方設定",
			siteKey:       "1x00000000000000000000AA",
			secretKey:     "1x0000000000000000000000000000000AA",
			wantSiteKey:   "1x00000000000000000000AA",
			wantSecretKey: "1x0000000000000000000000000000000AA",
		},
		{
			name:          "未設定",
			siteKey:       "",
			secretKey:     "",
			wantSiteKey:   "",
			wantSecretKey: "",
		},
		{
			name:          "Site Keyのみ設定",
			siteKey:       "1x00000000000000000000AA",
			secretKey:     "",
			wantSiteKey:   "1x00000000000000000000AA",
			wantSecretKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnv(t)
			defer cleanup()

			if tt.siteKey != "" {
				_ = os.Setenv("WIKINO_TURNSTILE_SITE_KEY", tt.siteKey)
			} else {
				_ = os.Unsetenv("WIKINO_TURNSTILE_SITE_KEY")
			}
			if tt.secretKey != "" {
				_ = os.Setenv("WIKINO_TURNSTILE_SECRET_KEY", tt.secretKey)
			} else {
				_ = os.Unsetenv("WIKINO_TURNSTILE_SECRET_KEY")
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() failed: %v", err)
			}

			if cfg.TurnstileSiteKey != tt.wantSiteKey {
				t.Errorf("TurnstileSiteKey = %q, want %q", cfg.TurnstileSiteKey, tt.wantSiteKey)
			}
			if cfg.TurnstileSecretKey != tt.wantSecretKey {
				t.Errorf("TurnstileSecretKey = %q, want %q", cfg.TurnstileSecretKey, tt.wantSecretKey)
			}
		})
	}
}

// TestLoad_MaintenanceMode は メンテナンスモード設定のテスト
func TestLoad_MaintenanceMode(t *testing.T) {
	tests := []struct {
		name                string
		maintenanceMode     string
		adminIP             string
		wantMaintenanceMode bool
		wantAdminIPs        []string
	}{
		{
			name:                "メンテナンスモードON、単一IP",
			maintenanceMode:     "on",
			adminIP:             "192.168.1.1",
			wantMaintenanceMode: true,
			wantAdminIPs:        []string{"192.168.1.1"},
		},
		{
			name:                "メンテナンスモードON、複数IP",
			maintenanceMode:     "on",
			adminIP:             "192.168.1.1,10.0.0.1",
			wantMaintenanceMode: true,
			wantAdminIPs:        []string{"192.168.1.1", "10.0.0.1"},
		},
		{
			name:                "メンテナンスモードOFF",
			maintenanceMode:     "off",
			adminIP:             "192.168.1.1",
			wantMaintenanceMode: false,
			wantAdminIPs:        []string{"192.168.1.1"},
		},
		{
			name:                "メンテナンスモード未設定",
			maintenanceMode:     "",
			adminIP:             "",
			wantMaintenanceMode: false,
			wantAdminIPs:        nil,
		},
		{
			name:                "メンテナンスモードON、管理者IP未設定",
			maintenanceMode:     "on",
			adminIP:             "",
			wantMaintenanceMode: true,
			wantAdminIPs:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnv(t)
			defer cleanup()

			if tt.maintenanceMode != "" {
				_ = os.Setenv("WIKINO_MAINTENANCE_MODE", tt.maintenanceMode)
			} else {
				_ = os.Unsetenv("WIKINO_MAINTENANCE_MODE")
			}
			if tt.adminIP != "" {
				_ = os.Setenv("WIKINO_ADMIN_IP", tt.adminIP)
			} else {
				_ = os.Unsetenv("WIKINO_ADMIN_IP")
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() failed: %v", err)
			}

			if cfg.MaintenanceMode != tt.wantMaintenanceMode {
				t.Errorf("MaintenanceMode = %v, want %v", cfg.MaintenanceMode, tt.wantMaintenanceMode)
			}
			if !reflect.DeepEqual(cfg.AdminIPs, tt.wantAdminIPs) {
				t.Errorf("AdminIPs = %v, want %v", cfg.AdminIPs, tt.wantAdminIPs)
			}
		})
	}
}

// TestLoad_DisableRateLimit は DisableRateLimit 設定のテスト
func TestLoad_DisableRateLimit(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"true", "true", true},
		{"false", "false", false},
		{"未設定", "", false},
		{"other", "yes", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnv(t)
			defer cleanup()

			if tt.value != "" {
				_ = os.Setenv("WIKINO_DISABLE_RATE_LIMIT", tt.value)
			} else {
				_ = os.Unsetenv("WIKINO_DISABLE_RATE_LIMIT")
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() failed: %v", err)
			}

			if cfg.DisableRateLimit != tt.want {
				t.Errorf("DisableRateLimit = %v, want %v", cfg.DisableRateLimit, tt.want)
			}
		})
	}
}

// TestLoad_RailsAppURL は RailsAppURL 設定のテスト
func TestLoad_RailsAppURL(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	railsURL := "http://localhost:3001"
	_ = os.Setenv("WIKINO_RAILS_APP_URL", railsURL)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.RailsAppURL != railsURL {
		t.Errorf("RailsAppURL = %v, want %v", cfg.RailsAppURL, railsURL)
	}
}

// TestParseAdminIPs は parseAdminIPs 関数のテスト
func TestParseAdminIPs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "単一IP",
			input: "192.168.1.1",
			want:  []string{"192.168.1.1"},
		},
		{
			name:  "複数IP",
			input: "192.168.1.1,10.0.0.1",
			want:  []string{"192.168.1.1", "10.0.0.1"},
		},
		{
			name:  "複数IPスペースあり",
			input: "192.168.1.1, 10.0.0.1, 172.16.0.1",
			want:  []string{"192.168.1.1", "10.0.0.1", "172.16.0.1"},
		},
		{
			name:  "空白のみの要素を除去",
			input: "192.168.1.1,  ,10.0.0.1",
			want:  []string{"192.168.1.1", "10.0.0.1"},
		},
		{
			name:  "空文字列",
			input: "",
			want:  []string{},
		},
		{
			name:  "先頭と末尾の空白を除去",
			input: "  192.168.1.1  ",
			want:  []string{"192.168.1.1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAdminIPs(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseAdminIPs(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestLoad_SentryConfig は Sentry 環境変数の読み込みをテスト
func TestLoad_SentryConfig(t *testing.T) {
	tests := []struct {
		name                 string
		dsn                  string
		environment          string
		tracesSampleRate     string
		debug                string
		wantDSN              string
		wantEnvironment      string
		wantTracesSampleRate float64
		wantDebug            bool
	}{
		{
			name:                 "全て未設定 (Sentry 無効化、Environment は APP_ENV にフォールバック)",
			dsn:                  "",
			environment:          "",
			tracesSampleRate:     "",
			debug:                "",
			wantDSN:              "",
			wantEnvironment:      "test", // setupTestEnv で APP_ENV=test に設定済み
			wantTracesSampleRate: 0.5,
			wantDebug:            false,
		},
		{
			name:                 "全て設定",
			dsn:                  "https://example@o0.ingest.sentry.io/0",
			environment:          "prod",
			tracesSampleRate:     "0.25",
			debug:                "true",
			wantDSN:              "https://example@o0.ingest.sentry.io/0",
			wantEnvironment:      "prod",
			wantTracesSampleRate: 0.25,
			wantDebug:            true,
		},
		{
			name:                 "DSN のみ設定 (他はデフォルト)",
			dsn:                  "https://example@o0.ingest.sentry.io/0",
			environment:          "",
			tracesSampleRate:     "",
			debug:                "",
			wantDSN:              "https://example@o0.ingest.sentry.io/0",
			wantEnvironment:      "test",
			wantTracesSampleRate: 0.5,
			wantDebug:            false,
		},
		{
			name:                 "Debug が true 以外の値は false 扱い",
			dsn:                  "",
			environment:          "",
			tracesSampleRate:     "",
			debug:                "yes",
			wantDSN:              "",
			wantEnvironment:      "test",
			wantTracesSampleRate: 0.5,
			wantDebug:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnv(t)
			defer cleanup()

			if tt.dsn != "" {
				_ = os.Setenv("WIKINO_SENTRY_DSN", tt.dsn)
			}
			if tt.environment != "" {
				_ = os.Setenv("WIKINO_SENTRY_ENVIRONMENT", tt.environment)
			}
			if tt.tracesSampleRate != "" {
				_ = os.Setenv("WIKINO_SENTRY_TRACES_SAMPLE_RATE", tt.tracesSampleRate)
			}
			if tt.debug != "" {
				_ = os.Setenv("WIKINO_SENTRY_DEBUG", tt.debug)
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() failed: %v", err)
			}

			if cfg.SentryDSN != tt.wantDSN {
				t.Errorf("SentryDSN = %q, want %q", cfg.SentryDSN, tt.wantDSN)
			}
			if cfg.SentryEnvironment != tt.wantEnvironment {
				t.Errorf("SentryEnvironment = %q, want %q", cfg.SentryEnvironment, tt.wantEnvironment)
			}
			if cfg.SentryTracesSampleRate != tt.wantTracesSampleRate {
				t.Errorf("SentryTracesSampleRate = %v, want %v", cfg.SentryTracesSampleRate, tt.wantTracesSampleRate)
			}
			if cfg.SentryDebug != tt.wantDebug {
				t.Errorf("SentryDebug = %v, want %v", cfg.SentryDebug, tt.wantDebug)
			}
		})
	}
}

// TestLoad_SentryEnvironment_FallbackToAppEnv は SentryEnvironment が APP_ENV にフォールバックすることをテスト
func TestLoad_SentryEnvironment_FallbackToAppEnv(t *testing.T) {
	tests := []struct {
		name    string
		appEnv  string
		wantEnv string
	}{
		{name: "dev へフォールバック", appEnv: "dev", wantEnv: "dev"},
		{name: "test へフォールバック", appEnv: "test", wantEnv: "test"},
		{name: "prod へフォールバック", appEnv: "prod", wantEnv: "prod"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnv(t)
			defer cleanup()

			_ = os.Setenv("APP_ENV", tt.appEnv)
			_ = os.Unsetenv("WIKINO_SENTRY_ENVIRONMENT")

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() failed: %v", err)
			}

			if cfg.SentryEnvironment != tt.wantEnv {
				t.Errorf("SentryEnvironment = %q, want %q", cfg.SentryEnvironment, tt.wantEnv)
			}
		})
	}
}

// TestParseSentryTracesSampleRate は parseSentryTracesSampleRate 関数のテスト
func TestParseSentryTracesSampleRate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  float64
	}{
		{name: "空文字列はデフォルト 0.5", input: "", want: 0.5},
		{name: "下限 0.0", input: "0.0", want: 0.0},
		{name: "中間値 0.5", input: "0.5", want: 0.5},
		{name: "0.25 を受理", input: "0.25", want: 0.25},
		{name: "上限 1.0", input: "1.0", want: 1.0},
		{name: "範囲外 (負数) はデフォルト 0.5", input: "-0.1", want: 0.5},
		{name: "範囲外 (1.0 超過) はデフォルト 0.5", input: "1.5", want: 0.5},
		{name: "パース不可文字列はデフォルト 0.5", input: "abc", want: 0.5},
		{name: "整数表記も許容", input: "1", want: 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSentryTracesSampleRate(tt.input)
			if got != tt.want {
				t.Errorf("parseSentryTracesSampleRate(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestGetAssetVersion は GetAssetVersion メソッドのテスト
func TestGetAssetVersion(t *testing.T) {
	tests := []struct {
		name         string
		env          string
		assetVersion string
		wantStatic   bool
	}{
		{
			name:         "開発環境では動的値",
			env:          "dev",
			assetVersion: "abc123",
			wantStatic:   false,
		},
		{
			name:         "テスト環境ではGitハッシュ",
			env:          "test",
			assetVersion: "abc123",
			wantStatic:   true,
		},
		{
			name:         "本番環境ではGitハッシュ",
			env:          "prod",
			assetVersion: "abc123",
			wantStatic:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Env: tt.env, AssetVersion: tt.assetVersion}

			got1 := cfg.GetAssetVersion()
			got2 := cfg.GetAssetVersion()

			if tt.wantStatic {
				// 静的値（AssetVersion）が返されるべき
				if got1 != tt.assetVersion {
					t.Errorf("GetAssetVersion() = %v, want %v", got1, tt.assetVersion)
				}
				if got1 != got2 {
					t.Errorf("GetAssetVersion() should return same value, got %v and %v", got1, got2)
				}
			} else {
				// 動的値（タイムスタンプ）が返されるべき
				if got1 == "" {
					t.Error("GetAssetVersion() should not return empty string")
				}
			}
		})
	}
}

// TestGetGitCommitHash verifies that the GIT_REV environment variable takes
// precedence and is shortened to 7 characters.
//
// A Dokku deploy target has no .git directory, so the git command fails;
// whether GIT_REV is usable therefore decides the Sentry release (avoiding a
// fallback to "dev").
//
// [Ja] GIT_REV 環境変数が最優先され、7 文字に短縮されることを検証する。
//
// Dokku のデプロイ先には .git が無く git コマンドが失敗するため、GIT_REV を
// 使えるかどうかが Sentry の release ("dev" 化の回避) を左右する。
func TestGetGitCommitHash(t *testing.T) {
	tests := []struct {
		name   string
		gitRev string
		want   string
	}{
		{
			name:   "フルSHAは7文字に短縮される",
			gitRev: "1234567890abcdef1234567890abcdef12345678",
			want:   "1234567",
		},
		{
			name:   "7文字以下ならそのまま返す",
			gitRev: "abc123",
			want:   "abc123",
		},
		{
			name:   "前後の空白は除去される",
			gitRev: "  1234567890abcdef  ",
			want:   "1234567",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GIT_REV", tt.gitRev)

			got := getGitCommitHash()
			if got != tt.want {
				t.Errorf("getGitCommitHash() = %q, want %q", got, tt.want)
			}
		})
	}
}

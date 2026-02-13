package middleware

import (
	"net/http"
	"time"

	"github.com/wikinoapp/wikino/go/internal/clientip"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/templates/pages/maintenance"
)

// MaintenanceMiddleware はメンテナンスモード時にアクセスを制限するミドルウェア
type MaintenanceMiddleware struct {
	cfg *config.Config
}

// NewMaintenanceMiddleware は新しいMaintenanceMiddlewareを作成します
func NewMaintenanceMiddleware(cfg *config.Config) *MaintenanceMiddleware {
	return &MaintenanceMiddleware{cfg: cfg}
}

// Middleware はHTTPミドルウェアを返します。
// メンテナンスモードが有効で、管理者IP以外からのアクセスの場合は503を返します。
// ヘルスチェックエンドポイントはメンテナンスモード中でも通常処理します。
func (m *MaintenanceMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.cfg.MaintenanceMode {
			next.ServeHTTP(w, r)
			return
		}

		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		if m.isAdminIP(r) {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Retry-After", time.Now().Add(1*time.Hour).Format(http.TimeFormat))
		w.WriteHeader(http.StatusServiceUnavailable)

		// テンプレートのレンダリングエラーはレスポンス書き込み後なので無視
		_ = maintenance.Page().Render(r.Context(), w)
	})
}

// isAdminIP はリクエスト元IPが管理者IPかどうかをチェックします
func (m *MaintenanceMiddleware) isAdminIP(r *http.Request) bool {
	ip := clientip.GetClientIP(r)
	for _, adminIP := range m.cfg.AdminIPs {
		if ip == adminIP {
			return true
		}
	}
	return false
}

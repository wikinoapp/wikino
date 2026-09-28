package middleware

import (
	"net/http"
	"time"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/clientip"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	errpages "github.com/wikinoapp/wikino/go/internal/templates/pages/errors"
)

// maintenanceRetryAfterはメンテナンス中のレスポンスで再開の見込みとして示す時間
const maintenanceRetryAfter = 1 * time.Hour

// MaintenanceMiddlewareはメンテナンスモード時にアクセスを制限するミドルウェア
type MaintenanceMiddleware struct {
	cfg      *config.Config
	problems *apierror.Writer
}

// NewMaintenanceMiddlewareは新しいMaintenanceMiddlewareを作成します
func NewMaintenanceMiddleware(cfg *config.Config, problems *apierror.Writer) *MaintenanceMiddleware {
	return &MaintenanceMiddleware{cfg: cfg, problems: problems}
}

// MiddlewareはHTTPミドルウェアを返します。
// メンテナンスモードが有効で、管理者IP以外からのアクセスの場合は503を返します。
// ヘルスチェックエンドポイントはメンテナンスモード中でも通常処理します。
// APIのパスには、HTMLのメンテナンスページではなくProblem Detailsの503を返します。
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

		if session.IsAPIPath(r.URL.Path) {
			m.problems.ServiceUnavailable(w, r, maintenanceRetryAfter)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Retry-After", time.Now().Add(maintenanceRetryAfter).Format(http.TimeFormat))
		w.WriteHeader(http.StatusServiceUnavailable)

		// テンプレートのレンダリングエラーはレスポンス書き込み後なので無視
		_ = errpages.MaintenancePage().Render(r.Context(), w)
	})
}

// isAdminIPはリクエスト元IPが管理者IPかどうかをチェックします
func (m *MaintenanceMiddleware) isAdminIP(r *http.Request) bool {
	ip := clientip.GetClientIP(r)
	for _, adminIP := range m.cfg.AdminIPs {
		if ip == adminIP {
			return true
		}
	}
	return false
}

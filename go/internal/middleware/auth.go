// Package middleware はHTTPミドルウェアを提供します
package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/session"
)

// contextKey はコンテキストに値を保存するための型
type contextKey string

const (
	// userContextKey はコンテキストにユーザー情報を保存するためのキー
	userContextKey contextKey = "user"
)

// Auth は認証関連のミドルウェアを提供する
type Auth struct {
	sessionMgr *session.Manager
}

// NewAuth は新しいAuthミドルウェアを作成する
func NewAuth(sessionMgr *session.Manager) *Auth {
	return &Auth{
		sessionMgr: sessionMgr,
	}
}

// RequireAuth protects routes that require authentication. An unauthenticated request is sent to
// the sign-in page.
//
// [Ja] RequireAuth は認証が必要なルートを保護するミドルウェア。
// 未認証の場合はサインインページにリダイレクトする。
func (a *Auth) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		user, err := a.sessionMgr.GetCurrentUser(ctx, r)
		if err != nil {
			slog.ErrorContext(ctx, "認証チェック中にエラーが発生", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if user == nil {
			redirectToSignIn(w, r)
			return
		}

		// コンテキストにユーザー情報を設定
		ctx = context.WithValue(ctx, userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// redirectToSignIn sends an unauthenticated visitor to the sign-in page. A GET or HEAD request
// carries its path and query in the back parameter so that signing in lands the visitor on the
// page they asked for. Other methods have side effects that signing in cannot replay with a GET,
// so they are sent to the sign-in page without it.
//
// [Ja] redirectToSignIn は未認証の訪問者をサインインページへ送る。
// GET と HEAD はパスとクエリを back パラメータに載せ、サインイン後に目的のページへ着けるようにする。
// それ以外のメソッドは副作用を持ち、サインイン後に GET で再現しても意味が無いため、
// back を付けずにサインインページへ送る。
func redirectToSignIn(w http.ResponseWriter, r *http.Request) {
	signInURL := "/sign_in"
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		signInURL += "?back=" + url.QueryEscape(r.URL.RequestURI())
	}

	http.Redirect(w, r, signInURL, http.StatusFound)
}

// RequireNoAuth は未認証が必要なルートを保護するミドルウェア
// 認証済みの場合はホームページにリダイレクトする
func (a *Auth) RequireNoAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		user, err := a.sessionMgr.GetCurrentUser(ctx, r)
		if err != nil {
			slog.ErrorContext(ctx, "認証チェック中にエラーが発生", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if user != nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// SetUser はコンテキストにユーザー情報を設定するミドルウェア
// RequireAuthとは異なり、認証チェックは行わず、ログインしていればユーザー情報を設定する
// 認証の有無に関わらずリクエストは処理される
func (a *Auth) SetUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		user, err := a.sessionMgr.GetCurrentUser(ctx, r)
		if err != nil {
			// エラーが発生してもリクエストは処理を継続
			slog.WarnContext(ctx, "ユーザー情報の取得に失敗", "error", err)
			next.ServeHTTP(w, r)
			return
		}

		if user != nil {
			ctx = context.WithValue(ctx, userContextKey, user)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserFromContext はコンテキストからユーザー情報を取得する
// ユーザーが設定されていない場合はnilを返す
func UserFromContext(ctx context.Context) *model.User {
	user, ok := ctx.Value(userContextKey).(*model.User)
	if !ok {
		return nil
	}
	return user
}

// SetUserToContext はコンテキストにユーザー情報を設定する
// テストでログイン状態をシミュレートするために使用する
func SetUserToContext(ctx context.Context, user *model.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

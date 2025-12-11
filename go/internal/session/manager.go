// Package session はセッション管理機能を提供します
package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// CookieName はセッショントークンを格納するCookieのキー名
// Rails版と同じキーを使用して、セッションを共有する
const CookieName = "user_session_tokens"

// Manager はセッション管理を行う構造体
type Manager struct {
	userRepo        *repository.UserRepository
	userSessionRepo *repository.UserSessionRepository
	cfg             *config.Config
}

// NewManager は Manager を生成する
func NewManager(
	userRepo *repository.UserRepository,
	userSessionRepo *repository.UserSessionRepository,
	cfg *config.Config,
) *Manager {
	return &Manager{
		userRepo:        userRepo,
		userSessionRepo: userSessionRepo,
		cfg:             cfg,
	}
}

// GetCurrentUser は現在ログインしているユーザーを取得する
// セッションが無効な場合は nil を返す
func (m *Manager) GetCurrentUser(ctx context.Context, r *http.Request) (*model.User, error) {
	token := m.getSessionToken(r)
	if token == "" {
		return nil, nil
	}

	session, err := m.userSessionRepo.FindByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil
	}

	user, err := m.userRepo.FindByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// SetSessionCookie はセッショントークンをCookieに設定する
func (m *Manager) SetSessionCookie(w http.ResponseWriter, token string) {
	cookie := &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		Domain:   m.cfg.CookieDomain,
		Secure:   m.cfg.SessionSecure,
		HttpOnly: m.cfg.SessionHTTPOnly,
		SameSite: http.SameSiteLaxMode,
		// Rails版と同じく長期間有効なCookieを設定（10年）
		MaxAge: 10 * 365 * 24 * 60 * 60,
	}
	http.SetCookie(w, cookie)
}

// DeleteSessionCookie はセッションCookieを削除する
func (m *Manager) DeleteSessionCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		Domain:   m.cfg.CookieDomain,
		Secure:   m.cfg.SessionSecure,
		HttpOnly: m.cfg.SessionHTTPOnly,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	http.SetCookie(w, cookie)
}

// getSessionToken はリクエストからセッショントークンを取得する
func (m *Manager) getSessionToken(r *http.Request) string {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// GenerateSecureToken は安全なセッショントークンを生成する
// Rails の has_secure_token と互換性のある形式で生成する
// 24バイトのランダムデータをBase64 URL-safeエンコードして32文字のトークンを生成
func GenerateSecureToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

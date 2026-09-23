// Package sessionはセッション管理機能を提供します
package session

import (
	"context"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// CookieNameはセッショントークンを格納するCookieのキー名
// Rails版と同じキーを使用して、セッションを共有する
const CookieName = "user_session_tokens"

// PendingUserCookieNameは2FA認証待ちユーザーIDを格納するCookieのキー名
const PendingUserCookieName = "pending_user_id"

// EmailConfirmationCookieNameはメール確認IDを格納するCookieのキー名
const EmailConfirmationCookieName = "email_confirmation_id"

// Managerはセッション管理を行う構造体
type Manager struct {
	userRepo        *repository.UserRepository
	userSessionRepo *repository.UserSessionRepository
	cfg             *config.Config
}

// NewManagerはManagerを生成する
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

// GetCurrentUserは現在ログインしているユーザーを取得する
// セッションが無効な場合はnilを返す
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

// SetSessionCookieはセッショントークンをCookieに設定する
func (m *Manager) SetSessionCookie(w http.ResponseWriter, token string) {
	cookie := &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		Domain:   m.cfg.CookieDomain,
		Secure:   m.cfg.SessionSecure,
		HttpOnly: m.cfg.SessionHTTPOnly,
		SameSite: http.SameSiteLaxMode,
		// Rails版と同じく長期間有効なCookieを設定 (10年)
		MaxAge: 10 * 365 * 24 * 60 * 60,
	}
	http.SetCookie(w, cookie)
}

// DeleteSessionCookieはセッションCookieを削除する
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

// getSessionTokenはリクエストからセッショントークンを取得する
func (m *Manager) getSessionToken(r *http.Request) string {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// GenerateSecureTokenはauth.GenerateSecureTokenのラッパー。
// 既存の呼び出し元 (middlewareなど) との互換性を維持する。
func GenerateSecureToken() (string, error) {
	return auth.GenerateSecureToken()
}

// SetPendingUserCookieは2FA認証待ちユーザーIDをCookieに設定する
func (m *Manager) SetPendingUserCookie(w http.ResponseWriter, userID model.UserID) {
	cookie := &http.Cookie{
		Name:     PendingUserCookieName,
		Value:    string(userID),
		Path:     "/",
		Domain:   m.cfg.CookieDomain,
		Secure:   m.cfg.SessionSecure,
		HttpOnly: m.cfg.SessionHTTPOnly,
		SameSite: http.SameSiteLaxMode,
		// 2FA認証のために一時的に設定 (10分間有効)
		MaxAge: 10 * 60,
	}
	http.SetCookie(w, cookie)
}

// GetPendingUserIDはCookieから2FA認証待ちユーザーIDを取得する
func (m *Manager) GetPendingUserID(r *http.Request) model.UserID {
	cookie, err := r.Cookie(PendingUserCookieName)
	if err != nil {
		return ""
	}
	return model.UserID(cookie.Value)
}

// DeletePendingUserCookieは2FA認証待ちユーザーIDのCookieを削除する
func (m *Manager) DeletePendingUserCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     PendingUserCookieName,
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

// SetEmailConfirmationCookieはメール確認IDをCookieに設定する
func (m *Manager) SetEmailConfirmationCookie(w http.ResponseWriter, emailConfirmationID string) {
	cookie := &http.Cookie{
		Name:     EmailConfirmationCookieName,
		Value:    emailConfirmationID,
		Path:     "/",
		Domain:   m.cfg.CookieDomain,
		Secure:   m.cfg.SessionSecure,
		HttpOnly: m.cfg.SessionHTTPOnly,
		SameSite: http.SameSiteLaxMode,
		// 確認コードの有効期限に合わせて15分間有効
		MaxAge: 15 * 60,
	}
	http.SetCookie(w, cookie)
}

// GetEmailConfirmationIDはCookieからメール確認IDを取得する
func (m *Manager) GetEmailConfirmationID(r *http.Request) string {
	cookie, err := r.Cookie(EmailConfirmationCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// DeleteEmailConfirmationCookieはメール確認IDのCookieを削除する
func (m *Manager) DeleteEmailConfirmationCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     EmailConfirmationCookieName,
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

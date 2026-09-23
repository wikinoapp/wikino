package session

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
)

// FlashCookieNameはフラッシュメッセージを格納するCookieのキー名
const FlashCookieName = "wikino_flash"

// FlashTypeはフラッシュメッセージの種類を表す
type FlashType string

const (
	// FlashSuccessは成功メッセージ
	FlashSuccess FlashType = "success"
	// FlashErrorはエラーメッセージ
	FlashError FlashType = "error"
	// FlashWarningは警告メッセージ
	FlashWarning FlashType = "warning"
	// FlashInfoは情報メッセージ
	FlashInfo FlashType = "info"
)

// FlashMessageはフラッシュメッセージを表す構造体
type FlashMessage struct {
	Type    FlashType `json:"type"`
	Message string    `json:"message"`
}

// FlashManagerはフラッシュメッセージを管理する構造体
type FlashManager struct {
	cookieDomain    string
	sessionSecure   bool
	sessionHTTPOnly bool
}

// NewFlashManagerはFlashManagerを生成する
func NewFlashManager(cookieDomain string, sessionSecure, sessionHTTPOnly bool) *FlashManager {
	return &FlashManager{
		cookieDomain:    cookieDomain,
		sessionSecure:   sessionSecure,
		sessionHTTPOnly: sessionHTTPOnly,
	}
}

// SetSuccessは成功メッセージを設定する
func (f *FlashManager) SetSuccess(w http.ResponseWriter, message string) {
	f.setFlash(w, FlashSuccess, message)
}

// SetErrorはエラーメッセージを設定する
func (f *FlashManager) SetError(w http.ResponseWriter, message string) {
	f.setFlash(w, FlashError, message)
}

// SetWarningは警告メッセージを設定する
func (f *FlashManager) SetWarning(w http.ResponseWriter, message string) {
	f.setFlash(w, FlashWarning, message)
}

// SetInfoは情報メッセージを設定する
func (f *FlashManager) SetInfo(w http.ResponseWriter, message string) {
	f.setFlash(w, FlashInfo, message)
}

// setFlashはフラッシュメッセージをCookieに設定する
func (f *FlashManager) setFlash(w http.ResponseWriter, flashType FlashType, message string) {
	flash := FlashMessage{
		Type:    flashType,
		Message: message,
	}
	data, err := json.Marshal(flash)
	if err != nil {
		return
	}

	// CookieのValueにはBase64エンコードして保存
	// JSONの特殊文字 (ダブルクォートなど) がCookieで無効な文字として扱われるため
	encoded := base64.StdEncoding.EncodeToString(data)

	cookie := &http.Cookie{
		Name:     FlashCookieName,
		Value:    encoded,
		Path:     "/",
		Domain:   f.cookieDomain,
		Secure:   f.sessionSecure,
		HttpOnly: false, // JavaScriptからアクセス可能にする
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}

// GetFlashはフラッシュメッセージを取得し、Cookieから削除する
func (f *FlashManager) GetFlash(w http.ResponseWriter, r *http.Request) *FlashMessage {
	cookie, err := r.Cookie(FlashCookieName)
	if err != nil {
		return nil
	}

	// Base64デコード
	data, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		f.clearFlash(w)
		return nil
	}

	var flash FlashMessage
	if err := json.Unmarshal(data, &flash); err != nil {
		// JSONパースに失敗した場合は無視
		f.clearFlash(w)
		return nil
	}

	// Cookieを削除
	f.clearFlash(w)

	return &flash
}

type flashContextKey struct{}

// Middlewareはリクエストからフラッシュメッセージを読み取り、contextに格納するミドルウェアです。
// Cookieからフラッシュを読み取り後、自動的にCookieを削除します。
func (f *FlashManager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flash := f.GetFlash(w, r)
		if flash != nil {
			ctx := context.WithValue(r.Context(), flashContextKey{}, flash)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

// FlashFromContextはcontextからフラッシュメッセージを取得します
func FlashFromContext(ctx context.Context) *FlashMessage {
	flash, _ := ctx.Value(flashContextKey{}).(*FlashMessage)
	return flash
}

// clearFlashはフラッシュCookieを削除する
func (f *FlashManager) clearFlash(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     FlashCookieName,
		Value:    "",
		Path:     "/",
		Domain:   f.cookieDomain,
		Secure:   f.sessionSecure,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	http.SetCookie(w, cookie)
}

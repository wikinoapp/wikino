package session

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
)

// FlashCookieName はフラッシュメッセージを格納するCookieのキー名
const FlashCookieName = "wikino_flash"

// FlashType はフラッシュメッセージの種類を表す
type FlashType string

const (
	// FlashNotice は成功・情報メッセージ
	FlashNotice FlashType = "notice"
	// FlashAlert は警告・エラーメッセージ
	FlashAlert FlashType = "alert"
)

// FlashMessage はフラッシュメッセージを表す構造体
type FlashMessage struct {
	Type    FlashType `json:"type"`
	Message string    `json:"message"`
}

// FlashManager はフラッシュメッセージを管理する構造体
type FlashManager struct {
	cookieDomain    string
	sessionSecure   bool
	sessionHTTPOnly bool
}

// NewFlashManager は FlashManager を生成する
func NewFlashManager(cookieDomain string, sessionSecure, sessionHTTPOnly bool) *FlashManager {
	return &FlashManager{
		cookieDomain:    cookieDomain,
		sessionSecure:   sessionSecure,
		sessionHTTPOnly: sessionHTTPOnly,
	}
}

// SetNotice は成功メッセージを設定する
func (f *FlashManager) SetNotice(w http.ResponseWriter, message string) {
	f.setFlash(w, FlashNotice, message)
}

// SetAlert は警告メッセージを設定する
func (f *FlashManager) SetAlert(w http.ResponseWriter, message string) {
	f.setFlash(w, FlashAlert, message)
}

// setFlash はフラッシュメッセージをCookieに設定する
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
	// JSONの特殊文字（ダブルクォートなど）がCookieで無効な文字として扱われるため
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

// GetFlash はフラッシュメッセージを取得し、Cookieから削除する
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

// clearFlash はフラッシュCookieを削除する
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

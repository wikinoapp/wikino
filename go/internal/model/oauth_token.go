package model

import (
	"errors"
)

// OAuthTokenErrorCodeは、トークン要求・トークンの失効・デバイス認可の要求の失敗をクライアントへ
// 伝えるエラーコード (RFC 6749 §5.2、RFC 7009 §2.2.1、RFC 8628 §3.2・§3.5、RFC 8707 §2)
type OAuthTokenErrorCode string

const (
	// OAuthTokenErrorInvalidRequestは必須のパラメーターの欠落・不正な値・重複、
	// または複数のクライアント認証の方式を同時に使ったことを表す
	OAuthTokenErrorInvalidRequest OAuthTokenErrorCode = "invalid_request"
	// OAuthTokenErrorInvalidClientはクライアントの認証の失敗を表す
	OAuthTokenErrorInvalidClient OAuthTokenErrorCode = "invalid_client"
	// OAuthTokenErrorUnauthorizedClientは、トークンの失効 (RFC 7009 §2.1) で、別のクライアントに
	// 発行されたトークンを失効しようとしたこと、またはデバイス認可を使えないクライアント
	// (confidentialクライアント) が要求したことを表す
	OAuthTokenErrorUnauthorizedClient OAuthTokenErrorCode = "unauthorized_client"
	// OAuthTokenErrorInvalidGrantは認可コード・リフレッシュトークン・デバイスコードが無効・期限切れ・
	// 使用済み・別のクライアントに発行されたもの、またはリダイレクトURI・PKCEの不一致を表す。
	// デバイスコードの期限切れは `expired_token` で表す
	OAuthTokenErrorInvalidGrant OAuthTokenErrorCode = "invalid_grant"
	// OAuthTokenErrorUnsupportedGrantTypeは、`authorization_code`・`refresh_token`・デバイスコード
	// (`urn:ietf:params:oauth:grant-type:device_code`) 以外のgrant_typeを表す
	OAuthTokenErrorUnsupportedGrantType OAuthTokenErrorCode = "unsupported_grant_type"
	// OAuthTokenErrorInvalidScopeは元の許可の範囲を超えるスコープ、またはデバイス認可の要求の
	// 不明な・トークンに付与できないスコープを表す
	OAuthTokenErrorInvalidScope OAuthTokenErrorCode = "invalid_scope"
	// OAuthTokenErrorInvalidTargetは `resource` の不正な値・トークンが属さないスペース・連携できない
	// スペースを表す
	OAuthTokenErrorInvalidTarget OAuthTokenErrorCode = "invalid_target"
	// OAuthTokenErrorAuthorizationPendingは、デバイスコードの交換で、利用者がまだ許可も拒否も
	// していないことを表す。クライアントはポーリングを続ける (RFC 8628 §3.5)
	OAuthTokenErrorAuthorizationPending OAuthTokenErrorCode = "authorization_pending"
	// OAuthTokenErrorSlowDownは、デバイスコードの交換で、許可待ちのままポーリングの間隔より早く
	// 届いたことを表す。クライアントは以降の間隔を5秒延ばす (RFC 8628 §3.5)
	OAuthTokenErrorSlowDown OAuthTokenErrorCode = "slow_down"
	// OAuthTokenErrorAccessDeniedは、デバイスコードの交換で、利用者が要求を拒否したことを表す (RFC 8628 §3.5)
	OAuthTokenErrorAccessDenied OAuthTokenErrorCode = "access_denied"
	// OAuthTokenErrorExpiredTokenは、デバイスコードの交換で、利用者が許可する前か交換する前に
	// デバイスコードの期限が切れたことを表す (RFC 8628 §3.5)
	OAuthTokenErrorExpiredToken OAuthTokenErrorCode = "expired_token"
)

// OAuthTokenErrorは、トークン要求・トークンの失効・デバイス認可の要求をエラーとしてクライアントへ
// 返すことを表す
type OAuthTokenError struct {
	Code OAuthTokenErrorCode
}

// Errorはerrorインターフェースを満たす
func (e *OAuthTokenError) Error() string {
	return "OAuthのトークン要求のエラー: " + string(e.Code)
}

// AsOAuthTokenErrorはerrがOAuthTokenErrorなら取り出し、そうでなければnilを返す
func AsOAuthTokenError(err error) *OAuthTokenError {
	var oe *OAuthTokenError
	if errors.As(err, &oe) {
		return oe
	}
	return nil
}

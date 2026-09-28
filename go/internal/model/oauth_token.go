package model

import (
	"errors"
)

// OAuthTokenErrorCodeは、トークン要求・トークンの失効の失敗をクライアントへ伝えるエラーコード
// (RFC 6749 §5.2、RFC 7009 §2.2.1、RFC 8707 §2)
type OAuthTokenErrorCode string

const (
	// OAuthTokenErrorInvalidRequestは必須のパラメーターの欠落・不正な値・重複、
	// または複数のクライアント認証の方式を同時に使ったことを表す
	OAuthTokenErrorInvalidRequest OAuthTokenErrorCode = "invalid_request"
	// OAuthTokenErrorInvalidClientはクライアントの認証の失敗を表す
	OAuthTokenErrorInvalidClient OAuthTokenErrorCode = "invalid_client"
	// OAuthTokenErrorUnauthorizedClientは、トークンの失効 (RFC 7009 §2.1) で、別のクライアントに
	// 発行されたトークンを失効しようとしたことを表す
	OAuthTokenErrorUnauthorizedClient OAuthTokenErrorCode = "unauthorized_client"
	// OAuthTokenErrorInvalidGrantは認可コード・リフレッシュトークンが無効・期限切れ・使用済み・
	// 別のクライアントに発行されたもの、またはリダイレクトURI・PKCEの不一致を表す
	OAuthTokenErrorInvalidGrant OAuthTokenErrorCode = "invalid_grant"
	// OAuthTokenErrorUnsupportedGrantTypeは `authorization_code`・`refresh_token` 以外のgrant_typeを表す
	OAuthTokenErrorUnsupportedGrantType OAuthTokenErrorCode = "unsupported_grant_type"
	// OAuthTokenErrorInvalidScopeは元の許可の範囲を超えるスコープを表す
	OAuthTokenErrorInvalidScope OAuthTokenErrorCode = "invalid_scope"
	// OAuthTokenErrorInvalidTargetは `resource` の不正な値・トークンの束縛先と異なるスペースを表す
	OAuthTokenErrorInvalidTarget OAuthTokenErrorCode = "invalid_target"
)

// OAuthTokenErrorは、トークン要求をエラーとしてクライアントへ返すことを表す
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

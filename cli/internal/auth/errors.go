package auth

import (
	"context"
	"errors"

	"github.com/wikinoapp/wikino/cli/internal/apiutil"
	"github.com/wikinoapp/wikino/cli/internal/config"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// ConvertOAuthErrorは、認可サーバーとの通信の失敗を、ヒントと終了コードを持つCLIのエラーにする。
// spaceはログインし直すコマンドに載せるスペースの識別子。
// 認可コード・コード検証子・トークンはエラーの値に含めていないため、メッセージにも出ない。
func ConvertOAuthError(loc *i18n.Localizer, space string, err error) error {
	var cliErr *output.Error
	if errors.As(err, &cliErr) {
		return cliErr
	}

	retryHint := loc.T("auth_error_oauth_retry_hint", map[string]any{"Space": space})
	newError := func(code, messageID string, data map[string]any, hint string, exitCode int) *output.Error {
		return &output.Error{Code: code, Message: loc.T(messageID, data), Hint: hint, ExitCode: exitCode, Err: err}
	}

	var oauthErr *OAuthError
	switch {
	case errors.Is(err, ErrStateMismatch):
		return newError(output.CodeOAuthError, "auth_error_state_mismatch", nil, retryHint, output.ExitAuth)
	case errors.Is(err, ErrIssuerMismatch):
		return newError(output.CodeOAuthError, "auth_error_issuer_mismatch", nil, retryHint, output.ExitAuth)
	case errors.Is(err, ErrLoopbackTimeout):
		return newError(output.CodeLoginTimeout, "auth_error_login_timeout", nil, retryHint, output.ExitError)
	case errors.Is(err, ErrDeviceCodeExpired):
		return newError(output.CodeLoginTimeout, "auth_error_device_expired", nil, retryHint, output.ExitError)
	case errors.Is(err, ErrDeviceAuthorizationUnsupported):
		return newError(output.CodeOAuthError, "auth_error_device_unsupported", nil, loc.T("auth_error_device_unsupported_hint"), output.ExitAuth)
	case errors.As(err, &oauthErr) && oauthErr.Code == "access_denied":
		return newError(output.CodeAuthorizationDenied, "auth_error_access_denied", nil, retryHint, output.ExitAuth)
	case errors.As(err, &oauthErr):
		return newError(output.CodeOAuthError, "auth_error_oauth", map[string]any{"Code": oauthErr.Code}, retryHint, output.ExitAuth)
	case errors.Is(err, ErrInvalidMetadata), errors.Is(err, ErrUnexpectedResponse):
		return newError(output.CodeResponse, "auth_error_oauth_response", nil, loc.T("auth_error_oauth_response_hint"), output.ExitError)
	}

	return apiutil.ConvertError(loc, err)
}

// convertRefreshErrorは、リフレッシュの失敗をCLIのエラーにする。
// リフレッシュトークンが拒否されたとき (期限切れ・失効・許可の取り消し) は、ログインし直しを促す。
func (s *Store) convertRefreshError(cred *config.Credential, err error) error {
	var oauthErr *OAuthError
	if errors.As(err, &oauthErr) && oauthErr.Code == "invalid_grant" {
		return &output.Error{
			Code:     output.CodeUnauthorized,
			Message:  s.Loc.T("auth_error_refresh_rejected", map[string]any{"Space": cred.Space, "Member": cred.MemberAtname}),
			Hint:     s.Loc.T("auth_error_oauth_retry_hint", map[string]any{"Space": cred.Space}),
			ExitCode: output.ExitAuth,
			Err:      err,
		}
	}

	return ConvertOAuthError(s.Loc, cred.Space, err)
}

// convertLockErrorは、資格情報のロックを待つ間の中止をCLIのエラーにする。それ以外のエラーはそのまま返す。
func (s *Store) convertLockError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return &output.Error{
			Code:     output.CodeUnexpected,
			Message:  s.Loc.T("auth_error_refresh_lock"),
			Hint:     s.Loc.T("auth_error_refresh_lock_hint"),
			ExitCode: output.ExitError,
			Err:      err,
		}
	}

	return err
}

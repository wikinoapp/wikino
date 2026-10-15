package apiutil

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/wikinoapp/wikino/cli/internal/apiclient"
	"github.com/wikinoapp/wikino/cli/internal/apigen"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// statusKindは、HTTPステータスごとのエラーのコード・終了コード・ヒントの翻訳キー。
type statusKind struct {
	code     string
	exitCode int
	hintID   string
}

var statusKinds = map[int]statusKind{
	// 400はリクエスト本文をJSONとして読めないときに返る。`wikino api --input` のように
	// 利用者が本文を渡す場面の入力の誤りのため、422と同じ終了コードにする。
	http.StatusBadRequest:           {output.CodeBadRequest, output.ExitValidation, "error_api_bad_request_hint"},
	http.StatusUnauthorized:         {output.CodeUnauthorized, output.ExitAuth, "error_api_unauthorized_hint"},
	http.StatusForbidden:            {output.CodeForbidden, output.ExitForbidden, "error_api_forbidden_hint"},
	http.StatusNotFound:             {output.CodeNotFound, output.ExitNotFound, "error_api_not_found_hint"},
	http.StatusConflict:             {output.CodeConflict, output.ExitConflict, "error_api_conflict_hint"},
	http.StatusPreconditionFailed:   {output.CodePreconditionFailed, output.ExitConflict, "error_api_precondition_failed_hint"},
	http.StatusPreconditionRequired: {output.CodePreconditionRequired, output.ExitConflict, "error_api_precondition_required_hint"},
	http.StatusUnprocessableEntity:  {output.CodeValidationFailed, output.ExitValidation, "error_api_validation_failed_hint"},
	http.StatusTooManyRequests:      {output.CodeRateLimited, output.ExitRateLimited, "error_api_rate_limited_hint"},
}

// ConvertErrorは、APIのクライアントが返したエラーを、ヒントと終了コードを持つCLIのエラーにする。
// 変換の対象でないエラー (CLIのエラーなど) はそのまま返す。
//
// ヒントには、ステータスごとに次に何をするかを書く。トークンなどの秘密の値は含めない。
// コマンドに固有のヒント (412で取り直すコマンドなど) が要る場合は、返したエラーのHintを上書きする。
func ConvertError(loc *i18n.Localizer, err error) error {
	if err == nil {
		return nil
	}

	var apiErr *apiclient.Error
	if errors.As(err, &apiErr) {
		return convertAPIError(loc, apiErr)
	}

	var responseErr *apigen.ResponseError
	if errors.As(err, &responseErr) {
		return &output.Error{
			Code:     output.CodeResponse,
			Message:  loc.T("error_response"),
			Hint:     loc.T("error_response_hint"),
			Status:   responseErr.StatusCode,
			ExitCode: output.ExitError,
			Err:      err,
		}
	}
	var inputErr *apiclient.InputError
	if errors.As(err, &inputErr) {
		return convertInputError(loc, inputErr)
	}

	var downloadErr *apiclient.DownloadError
	if errors.As(err, &downloadErr) {
		return &output.Error{
			Code:     output.CodeDownloadFailed,
			Message:  loc.T("error_download_failed", map[string]any{"Status": downloadErr.StatusCode}),
			Hint:     loc.T("error_download_failed_hint"),
			Status:   downloadErr.StatusCode,
			ExitCode: output.ExitError,
			Err:      err,
		}
	}
	if errors.Is(err, apiclient.ErrInsecureDownloadURL) {
		return &output.Error{
			Code:     output.CodeResponse,
			Message:  loc.T("error_insecure_download_url"),
			Hint:     loc.T("error_insecure_download_url_hint"),
			ExitCode: output.ExitError,
			Err:      err,
		}
	}

	// 原因のurl.Errorには署名が含まれうるため、その自由文を表示する分岐より先に扱う。
	// 署名付きURLのリダイレクトの拒否は、APIのリダイレクトの拒否とは案内が違うため、その分岐より先に見る。
	var downloadNetworkErr *apiclient.DownloadNetworkError
	if errors.As(err, &downloadNetworkErr) {
		if errors.Is(err, apiclient.ErrInsecureRedirect) {
			return &output.Error{
				Code:     output.CodeNetwork,
				Message:  loc.T("error_insecure_download_redirect"),
				Hint:     loc.T("error_insecure_download_url_hint"),
				ExitCode: output.ExitError,
				Err:      err,
			}
		}

		return &output.Error{
			Code:     output.CodeNetwork,
			Message:  loc.T("error_download_network"),
			Hint:     loc.T("error_download_failed_hint"),
			ExitCode: output.ExitError,
			Err:      err,
		}
	}

	// リダイレクトの拒否も `*url.Error` に包まれるため、通信の失敗より先に見る。
	// 最初の接続先には送信済みのため、ヒントは通信の失敗と同じく再送前の確認を案内する。
	if errors.Is(err, apiclient.ErrInsecureRedirect) {
		return &output.Error{
			Code:     output.CodeNetwork,
			Message:  loc.T("error_insecure_redirect"),
			Hint:     loc.T("error_network_hint"),
			ExitCode: output.ExitError,
			Err:      err,
		}
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return &output.Error{
			Code:     output.CodeNetwork,
			Message:  loc.T("error_network", map[string]any{"Detail": urlErr.Error()}),
			Hint:     loc.T("error_network_hint"),
			ExitCode: output.ExitError,
			Err:      err,
		}
	}

	return err
}

func convertAPIError(loc *i18n.Localizer, apiErr *apiclient.Error) *output.Error {
	kind, ok := statusKinds[apiErr.StatusCode]
	switch {
	case ok:
	case apiErr.StatusCode >= http.StatusInternalServerError:
		kind = statusKind{output.CodeServerError, output.ExitError, "error_api_server_error_hint"}
	default:
		kind = statusKind{output.CodeAPIError, output.ExitError, "error_api_error_hint"}
	}

	hint := loc.T(kind.hintID)
	switch {
	case apiErr.StatusCode == http.StatusForbidden && len(apiErr.RequiredScopes) > 0:
		hint = loc.T("error_api_forbidden_scope_hint", map[string]any{"Scopes": strings.Join(apiErr.RequiredScopes, " ")})
	case apiErr.StatusCode == http.StatusTooManyRequests && apiErr.HasRetryAfter:
		hint = loc.T("error_api_rate_limited_retry_after_hint", map[string]any{"Seconds": int(apiErr.RetryAfter.Seconds())})
	}

	return &output.Error{
		Code:     kind.code,
		Message:  apiErrorMessage(loc, apiErr),
		Hint:     hint,
		Status:   apiErr.StatusCode,
		ExitCode: kind.exitCode,
		Err:      apiErr,
	}
}

// apiErrorMessageは、Problem Detailsの説明 (サーバーがAccept-Languageの言語で返す) をメッセージにする。
// 入力の検証の問題は、どの項目のどんな問題かを読めるよう、同じ行に続けて並べる。
func apiErrorMessage(loc *i18n.Localizer, apiErr *apiclient.Error) string {
	message := apiErr.Detail
	if message == "" {
		message = apiErr.Title
	}
	if message == "" {
		message = loc.T("error_api_status", map[string]any{"Status": apiErr.StatusCode})
	}

	var details []string
	for _, fe := range apiErr.FieldErrors {
		switch {
		case fe.Pointer != "":
			details = append(details, fe.Pointer+": "+fe.Detail)
		case fe.Parameter != "":
			details = append(details, fe.Parameter+": "+fe.Detail)
		default:
			details = append(details, fe.Detail)
		}
	}
	if len(details) > 0 {
		message += " (" + strings.Join(details, "; ") + ")"
	}

	return message
}

// inputReasonIDsは、入力の値を受け付けない理由ごとの説明の翻訳キー。
var inputReasonIDs = map[apiclient.InputErrorReason]string{
	apiclient.ReasonEmpty:             "usage_error_invalid_value_empty",
	apiclient.ReasonControlCharacter:  "usage_error_invalid_value_control_character",
	apiclient.ReasonDotSegment:        "usage_error_invalid_value_dot_segment",
	apiclient.ReasonReservedCharacter: "usage_error_invalid_value_reserved_character",
	apiclient.ReasonInvalidFormat:     "usage_error_invalid_value_format",
	apiclient.ReasonInsecureScheme:    "usage_error_invalid_value_insecure_scheme",
	apiclient.ReasonNotAPIPath:        "usage_error_invalid_value_not_api_path",
}

func convertInputError(loc *i18n.Localizer, inputErr *apiclient.InputError) *output.Error {
	reasonID, ok := inputReasonIDs[inputErr.Reason]
	if !ok {
		reasonID = "usage_error_invalid_value_format"
	}

	return &output.Error{
		Code: output.CodeUsage,
		Message: loc.T("usage_error_invalid_value", map[string]any{
			"Name":   inputErr.Name,
			"Value":  inputErr.Value,
			"Reason": loc.T(reasonID),
		}),
		Hint:     loc.T("usage_error_invalid_value_hint"),
		ExitCode: output.ExitUsage,
		Err:      inputErr,
	}
}

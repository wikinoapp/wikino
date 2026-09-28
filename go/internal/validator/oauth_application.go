package validator

import (
	"context"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
)

const (
	// oauthApplicationNameMaxLengthはOAuthアプリ名の最大文字数
	oauthApplicationNameMaxLength = 50

	// oauthApplicationRedirectURIsMaxCountは1つのアプリに登録できるリダイレクトURIの最大数
	oauthApplicationRedirectURIsMaxCount = 10

	// oauthApplicationRedirectURIMaxLengthはリダイレクトURI1つの最大文字数
	oauthApplicationRedirectURIMaxLength = 2000
)

// OAuthApplicationCreateValidatorはOAuthアプリの登録のバリデーションを行う
type OAuthApplicationCreateValidator struct{}

// NewOAuthApplicationCreateValidatorはOAuthApplicationCreateValidatorを生成する
func NewOAuthApplicationCreateValidator() *OAuthApplicationCreateValidator {
	return &OAuthApplicationCreateValidator{}
}

// OAuthApplicationCreateValidatorInputはバリデーションの入力パラメータ。
// RedirectURIsとClientTypeはフォームが送信した文字列で、変換はバリデーターが行う。
// RedirectURIsは1行に1つのURIを書いたテキストである。
type OAuthApplicationCreateValidatorInput struct {
	Name         string
	RedirectURIs string
	ClientType   string
}

// OAuthApplicationCreateValidateOutputはバリデーション成功時の出力
type OAuthApplicationCreateValidateOutput struct {
	// RedirectURIsは登録するリダイレクトURI。入力の順を保ち、重複を除く
	RedirectURIs []string

	// ClientTypeはクライアントの種別
	ClientType model.OAuthClientType
}

// Validateはバリデーションを行い、フォームが送信したリダイレクトURIとクライアントの種別を
// 変換して返す。
func (v *OAuthApplicationCreateValidator) Validate(ctx context.Context, input OAuthApplicationCreateValidatorInput) (*OAuthApplicationCreateValidateOutput, error) {
	ve := model.NewValidationError()

	validateOAuthApplicationName(ctx, ve, input.Name)
	redirectURIs := validateOAuthApplicationRedirectURIs(ctx, ve, input.RedirectURIs)
	clientType := validateOAuthApplicationClientType(ctx, ve, input.ClientType)

	if ve.HasErrors() {
		return nil, ve
	}

	return &OAuthApplicationCreateValidateOutput{
		RedirectURIs: redirectURIs,
		ClientType:   clientType,
	}, nil
}

// OAuthApplicationUpdateValidatorはOAuthアプリの編集のバリデーションを行う
type OAuthApplicationUpdateValidator struct{}

// NewOAuthApplicationUpdateValidatorはOAuthApplicationUpdateValidatorを生成する
func NewOAuthApplicationUpdateValidator() *OAuthApplicationUpdateValidator {
	return &OAuthApplicationUpdateValidator{}
}

// OAuthApplicationUpdateValidatorInputはバリデーションの入力パラメータ。
// RedirectURIsは1行に1つのURIを書いたテキストである。クライアントの種別は編集できないため
// 受け取らない。
type OAuthApplicationUpdateValidatorInput struct {
	Name         string
	RedirectURIs string
}

// OAuthApplicationUpdateValidateOutputはバリデーション成功時の出力
type OAuthApplicationUpdateValidateOutput struct {
	// RedirectURIsは保存するリダイレクトURI。入力の順を保ち、重複を除く
	RedirectURIs []string
}

// Validateはバリデーションを行い、フォームが送信したリダイレクトURIを変換して返す。
// 名前とリダイレクトURIの規則は登録と同じである。
func (v *OAuthApplicationUpdateValidator) Validate(ctx context.Context, input OAuthApplicationUpdateValidatorInput) (*OAuthApplicationUpdateValidateOutput, error) {
	ve := model.NewValidationError()

	validateOAuthApplicationName(ctx, ve, input.Name)
	redirectURIs := validateOAuthApplicationRedirectURIs(ctx, ve, input.RedirectURIs)

	if ve.HasErrors() {
		return nil, ve
	}

	return &OAuthApplicationUpdateValidateOutput{RedirectURIs: redirectURIs}, nil
}

// validateOAuthApplicationNameはアプリ名の形式を検証し、見つかったものをveに積む。
// 名前は同意画面で利用者がアプリを見分けるためのもので、制御文字と先頭・末尾の空白を
// 拒否する理由はトピック名と同じである。
func validateOAuthApplicationName(ctx context.Context, ve *model.ValidationError, name string) {
	if name == "" {
		ve.AddField("name", i18n.T(ctx, "validation_oauth_application_name_required"))
		return
	}

	if utf8.RuneCountInString(name) > oauthApplicationNameMaxLength {
		ve.AddField("name", i18n.T(ctx, "validation_oauth_application_name_too_long"))
	}
	if strings.ContainsFunc(name, unicode.IsControl) {
		ve.AddField("name", i18n.T(ctx, "validation_oauth_application_name_control_chars"))
	}
	if strings.HasPrefix(name, " ") || strings.HasSuffix(name, " ") {
		ve.AddField("name", i18n.T(ctx, "validation_oauth_application_name_invalid_format"))
	}
}

// validateOAuthApplicationRedirectURIsは1行に1つ書かれたリダイレクトURIを検証し、登録する
// URIを返す。各行の前後の空白と空行は取り除くが、URIそのものは正規化しない。認可要求の
// リダイレクトURIは登録した文字列と完全一致で照合するため、利用者が書いた形のまま保存する。
func validateOAuthApplicationRedirectURIs(ctx context.Context, ve *model.ValidationError, value string) []string {
	var uris []string
	for line := range strings.Lines(value) {
		uri := strings.TrimSpace(line)
		if uri == "" || slices.Contains(uris, uri) {
			continue
		}
		uris = append(uris, uri)
	}

	if len(uris) == 0 {
		ve.AddField("redirect_uris", i18n.T(ctx, "validation_oauth_application_redirect_uris_required"))
		return nil
	}
	if len(uris) > oauthApplicationRedirectURIsMaxCount {
		ve.AddField("redirect_uris", i18n.T(ctx, "validation_oauth_application_redirect_uris_too_many", map[string]any{"Max": oauthApplicationRedirectURIsMaxCount}))
		return nil
	}

	for _, uri := range uris {
		if key := redirectURIErrorKey(uri); key != "" {
			ve.AddField("redirect_uris", i18n.T(ctx, key, map[string]any{"URI": uri}))
		}
	}
	return uris
}

// redirectURIErrorKeyはリダイレクトURIとして受け付けられない理由の翻訳キーを返す。
// 受け付けられる場合は空文字列を返す。
//
// RFC 9700 (OAuth 2.0 Security BCP) に沿って、認可コードを平文で運ばないようHTTPSに限る。
// ネイティブアプリがループバックで受け取る場合 (RFC 8252 §7.3) だけHTTPを許し、そのホストは
// IPアドレスのリテラル (127.0.0.1と[::1]) に限る。localhostは名前解決の設定次第で外部の
// ホストを指しうるため、RFC 8252 §8.3が推奨していない。フラグメントはRFC 6749 §3.1.2が禁じている。
func redirectURIErrorKey(uri string) string {
	if utf8.RuneCountInString(uri) > oauthApplicationRedirectURIMaxLength {
		return "validation_oauth_application_redirect_uri_too_long"
	}
	if strings.ContainsFunc(uri, unicode.IsSpace) {
		return "validation_oauth_application_redirect_uri_invalid"
	}

	u, err := url.Parse(uri)
	if err != nil || u.Opaque != "" || u.Host == "" || u.User != nil {
		return "validation_oauth_application_redirect_uri_invalid"
	}
	if strings.Contains(uri, "#") {
		return "validation_oauth_application_redirect_uri_fragment"
	}

	switch u.Scheme {
	case "https":
		return ""
	case "http":
		if host := u.Hostname(); host == "127.0.0.1" || host == "::1" {
			return ""
		}
		return "validation_oauth_application_redirect_uri_insecure"
	default:
		return "validation_oauth_application_redirect_uri_insecure"
	}
}

// validateOAuthApplicationClientTypeは送信されたクライアントの種別を検証して返す。
// 選択肢 (model.OAuthClientTypes) 以外の値は拒否する。
func validateOAuthApplicationClientType(ctx context.Context, ve *model.ValidationError, value string) model.OAuthClientType {
	for _, clientType := range model.OAuthClientTypes {
		if value == clientType.String() {
			return clientType
		}
	}
	ve.AddField("client_type", i18n.T(ctx, "validation_oauth_application_client_type_invalid"))
	return 0
}

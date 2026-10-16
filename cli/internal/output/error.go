package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/wikinoapp/wikino/cli/internal/i18n"
)

// 終了コード。エージェントがメッセージを読まずに失敗の種類を見分けられるよう、種類ごとに分ける。
// 使い方の誤りを2、分類しない失敗を1とするのは、Goツールチェインやgetoptの慣習に従う。
const (
	ExitOK                   = 0
	ExitError                = 1
	ExitUsage                = 2
	ExitAuth                 = 3
	ExitForbidden            = 4
	ExitNotFound             = 5
	ExitConflict             = 6
	ExitValidation           = 7
	ExitRateLimited          = 8
	ExitConfirmationRequired = 9
	// ExitInterruptedはCtrl-Cで中断されたときの終了コード。CLIはSIGINTを捕まえず、
	// シェルがシグナルによる終了を128+2として報告する値を表に載せるために定義する。
	ExitInterrupted = 130
)

// exitCodesは終了コードの表で、ヘルプに載せる順に並べる。
var exitCodes = []struct {
	code      int
	messageID string
}{
	{ExitOK, "exit_code_ok"},
	{ExitError, "exit_code_error"},
	{ExitUsage, "exit_code_usage"},
	{ExitAuth, "exit_code_auth"},
	{ExitForbidden, "exit_code_forbidden"},
	{ExitNotFound, "exit_code_not_found"},
	{ExitConflict, "exit_code_conflict"},
	{ExitValidation, "exit_code_validation"},
	{ExitRateLimited, "exit_code_rate_limited"},
	{ExitConfirmationRequired, "exit_code_confirmation_required"},
	{ExitInterrupted, "exit_code_interrupted"},
}

// ExitCodeTableは、終了コードとその意味の表をヘルプの節の本文として返す。
func ExitCodeTable(loc *i18n.Localizer) string {
	var b strings.Builder
	for i, e := range exitCodes {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "  %-4d %s", e.code, loc.T(e.messageID))
	}

	return b.String()
}

// エラーのコード。JSONの `code` に出す機械可読の分類で、公開の契約として英語のまま固定する。
const (
	CodeUsage                = "usage_error"
	CodeUnexpected           = "unexpected_error"
	CodeJQFailed             = "jq_failed"
	CodeConfirmationDeclined = "confirmation_declined"
	CodeNetwork              = "network_error"
	CodeResponse             = "response_error"
	CodeNotLoggedIn          = "not_logged_in"
	// OAuthのログインの失敗。認可サーバーのエラー・stateやissの不一致、利用者の拒否、待ち時間の超過を分ける。
	CodeOAuthError          = "oauth_error"
	CodeAuthorizationDenied = "authorization_denied"
	CodeLoginTimeout        = "login_timeout"
	// 添付ファイルの取得 (`attachment get`) の失敗。署名付きURLからの取得の失敗と、保存先に既にファイルがあることと、
	// 保存先に書き込めないことを分ける。
	CodeDownloadFailed = "download_failed"
	CodeFileExists     = "file_exists"
	CodeWriteFailed    = "write_failed"
)

// APIの応答に由来するエラーのコード。HTTPステータスごとに決める。
const (
	CodeBadRequest           = "bad_request"
	CodeUnauthorized         = "unauthorized"
	CodeForbidden            = "forbidden"
	CodeNotFound             = "not_found"
	CodeConflict             = "conflict"
	CodePreconditionFailed   = "precondition_failed"
	CodePreconditionRequired = "precondition_required"
	CodeValidationFailed     = "validation_failed"
	CodeRateLimited          = "rate_limited"
	CodeServerError          = "server_error"
	CodeAPIError             = "api_error"
)

// ErrorはCLIのエラー。何が起きたか (Message) と次に何をするか (Hint) を持ち、
// 終了コードとJSONの `code` で失敗の種類を表す。
//
// 値を組み合わせる場面ごとに必要な項目が違うため、コンストラクタを揃えず構造体リテラルで作る。
type Error struct {
	// Codeは機械可読の分類 (usage_errorなど)。
	Code string
	// Messageは何が起きたかを表す、利用者に見せる文言。
	Message string
	// Hintは次に何をするかを表す。実在するコマンドを書き、推測で補正したコマンドは示さない。
	// トークンなどの秘密の値を含めてはならない。
	Hint string
	// StatusはAPIの応答のHTTPステータス。APIに由来しないエラーでは0。
	Status int
	// ExitCodeはプロセスの終了コード。
	ExitCode int
	// Errは原因のエラー。利用者には出さない。
	Err error
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

// SilentErrorは、出力を済ませてから終了コードだけを返すエラー。
// 確認プロトコルのように、標準出力に結果を出したうえで成功と区別させたい場合に使う。
type SilentError struct {
	ExitCode int
}

func (e *SilentError) Error() string {
	return fmt.Sprintf("exit status %d", e.ExitCode)
}

// ExitCodeOfは、errに対応する終了コードを返す。
func ExitCodeOf(err error) int {
	if err == nil {
		return ExitOK
	}

	var silent *SilentError
	if errors.As(err, &silent) {
		return silent.ExitCode
	}

	var cliErr *Error
	if errors.As(err, &cliErr) {
		return cliErr.ExitCode
	}

	return ExitError
}

// jsonErrorはエラーのJSONの形。
type jsonError struct {
	Error jsonErrorBody `json:"error"`
}

type jsonErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
	Status  int    `json:"status,omitempty"`
}

// WriteErrorは、errを標準エラーなどのwに出す。asJSONならJSONを1行で、そうでなければ
// メッセージとヒントを1行ずつ出す。SilentErrorは既に出力を済ませているため何も出さない。
func WriteError(w io.Writer, err error, asJSON bool) error {
	if err == nil {
		return nil
	}

	var silent *SilentError
	if errors.As(err, &silent) {
		return nil
	}

	var cliErr *Error
	if !errors.As(err, &cliErr) {
		cliErr = &Error{Code: CodeUnexpected, Message: err.Error(), ExitCode: ExitError, Err: err}
	}

	if !asJSON {
		// メッセージには `--jq` のエラー文やAPIの応答など外部由来の値が入りうるため、
		// 端末を操作する制御文字を空白に置き換える。JSONはエスケープされるため元の値のまま出す。
		text := SanitizeLine(cliErr.Message) + "\n"
		if cliErr.Hint != "" {
			text += SanitizeLine(cliErr.Hint) + "\n"
		}
		_, werr := io.WriteString(w, text)

		return werr
	}

	var buf bytes.Buffer
	if err := encodeJSON(&buf, jsonError{Error: jsonErrorBody{
		Code:    cliErr.Code,
		Message: cliErr.Message,
		Hint:    cliErr.Hint,
		Status:  cliErr.Status,
	}}, false); err != nil {
		return err
	}
	_, werr := buf.WriteTo(w)

	return werr
}

// encodeJSONはvをJSONにしてwに書く。indentなら字下げし、そうでなければ1行にする。
// 本文やタイトルの "<" ">" "&" を < のような形にせず、そのまま読めるようにする。
func encodeJSON(w io.Writer, v any, indent bool) error {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if indent {
		encoder.SetIndent("", "  ")
	}
	if err := encoder.Encode(v); err != nil {
		return err
	}

	_, err := w.Write(escapeC1(buf.Bytes()))

	return err
}

// escapeC1は、JSONの中のC1制御文字 (U+0080〜U+009F) を `\u0080` の形にエスケープする。
// encoding/jsonはC1を生のバイトで書くが、端末によってはエスケープシーケンスの開始 (U+009BのCSIなど) と
// 解釈される。エスケープしてもデコードした値は変わらない。
//
// encoding/jsonの出力は正しいUTF-8で、C1は必ず0xC2と0x80〜0x9Fの2バイトになる。
// 0xC2は他の文字の途中に現れないため、バイト列のまま置き換えられる。
func escapeC1(data []byte) []byte {
	if !bytes.Contains(data, []byte{0xc2}) {
		return data
	}

	escaped := make([]byte, 0, len(data))
	for i := 0; i < len(data); i++ {
		if data[i] == 0xc2 && i+1 < len(data) && data[i+1] >= 0x80 && data[i+1] <= 0x9f {
			escaped = fmt.Appendf(escaped, `\u%04x`, data[i+1])
			i++
			continue
		}
		escaped = append(escaped, data[i])
	}

	return escaped
}

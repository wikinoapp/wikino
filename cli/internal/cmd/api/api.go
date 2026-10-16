// Package apiは、任意の公開APIを認証付きで呼ぶ `wikino api` のコマンドを提供する。
package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/apiclient"
	"github.com/wikinoapp/wikino/cli/internal/apigen"
	"github.com/wikinoapp/wikino/cli/internal/apiutil"
	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// spacePlaceholderは、パスの中で、コマンドを実行するスペースの識別子に置き換える文字列。
// 既定のスペースや `WIKINO_SPACE` で決まるスペースの識別子を、エージェントが調べずに書けるようにする。
const spacePlaceholder = "{space}"

// pathArgNameは、エラーに載せるパスの引数の名前。
const pathArgName = "<path>"

// methodsは、送れるHTTPメソッド。公開APIのoperationが使うものに限る。
var methods = []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

// forbiddenHeadersは、`--header` で指定させないヘッダー。
// 秘密の値はシェルの履歴やプロセスの一覧に残るフラグで受け取らず、接続先は `--host` で決めるため。
var forbiddenHeaders = []string{"Authorization", "Proxy-Authorization", "Cookie", "Host"}

// headerNameRegexpは、ヘッダーの名前に使える文字 (RFC 9110のtoken)。
var headerNameRegexp = regexp.MustCompile("^[!#$%&'*+.^_`|~0-9A-Za-z-]+$")

type options struct {
	input   string
	headers []string
	dryRun  *bool
	yes     *bool
}

// NewCmdAPIは `wikino api` を返す。
func NewCmdAPI(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	opts := &options{}
	cmd := &cobra.Command{
		Use:     "api <method> <path>",
		Short:   loc.T("api_short"),
		Long:    loc.T("api_long"),
		Example: loc.T("api_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("api_output"),
			cmdutil.AnnotationErrors: loc.T("api_errors"),
		},
		Args: cmdutil.ExactArgs(loc, 2),
	}
	cmd.Flags().StringVar(&opts.input, "input", "", loc.T("api_input_flag"))
	cmd.Flags().StringArrayVarP(&opts.headers, "header", "H", nil, loc.T("api_header_flag"))
	opts.dryRun = cmdutil.AddDryRunFlag(cmd, loc)
	opts.yes = cmdutil.AddYesFlag(cmd, loc)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return run(f, cmd, opts, args[0], args[1])
	}

	return cmd
}

func run(f *cmdutil.Factory, cmd *cobra.Command, opts *options, methodArg, pathArg string) error {
	loc := f.Localizer

	method := strings.ToUpper(methodArg)
	if !slices.Contains(methods, method) {
		return cmdutil.NewUsageError(loc, cmd, loc.T("api_error_invalid_method", map[string]any{
			"Value": methodArg, "Methods": strings.Join(methods, ", "),
		}))
	}

	// スペースが決まる前に、パスの形だけを検査する。識別子の文字種のダミーで置き換えても、
	// 当たるoperationは変わらない。
	draft, err := apiclient.ParseAPIPath(pathArgName, strings.ReplaceAll(pathArg, spacePlaceholder, "space"))
	if err != nil {
		return apiutil.ConvertError(loc, err)
	}

	header, err := parseHeaders(f, cmd, opts.headers)
	if err != nil {
		return err
	}

	// 応答はAPIの形のまま出すため、項目を選べない。絞り込みは `--jq` で行う。
	if cmd.Flags().Changed(output.FlagFields) {
		return cmdutil.NewUsageError(loc, cmd, loc.T("api_error_fields_unsupported"))
	}
	if opts.input != "" && method == http.MethodGet {
		return cmdutil.NewUsageError(loc, cmd, loc.T("api_error_input_with_get"))
	}

	op, matched := apiclient.MatchOperation(method, draft.EscapedPath())
	needsConfirm := matched && op.ConfirmationRequired && !*opts.dryRun && !*opts.yes
	// 承認後のコマンドで同じ本文を渡し直せず、対話の確認でも標準入力を使うため、併用させない。
	if needsConfirm && opts.input == "-" {
		return cmdutil.NewUsageError(loc, cmd, loc.T("api_error_stdin_with_confirmation", map[string]any{"Operation": op.ID}))
	}

	printer, err := output.NewPrinter(f.IOStreams, loc, cmd.Flags(), output.Spec{})
	if err != nil {
		return err
	}

	body, err := readInput(f, cmd, opts.input)
	if err != nil {
		return err
	}
	if body != nil && header.Get("Content-Type") == "" {
		header.Set("Content-Type", defaultContentType(method))
	}

	var dryRun *apiutil.DryRun
	if *opts.dryRun {
		dryRun = &apiutil.DryRun{IncludeReadOnly: true}
	}
	client, space, err := f.APIClientWithDryRun(cmd, dryRun)
	if err != nil {
		return err
	}

	ref, err := apiclient.ParseAPIPath(pathArgName, strings.ReplaceAll(pathArg, spacePlaceholder, url.PathEscape(space)))
	if err != nil {
		return apiutil.ConvertError(loc, err)
	}

	if needsConfirm {
		if err := output.Confirm(f.IOStreams, loc, confirmation(f, cmd, opts, method, ref, op)); err != nil {
			return err
		}
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	resp, err := client.Do(cmd.Context(), method, ref.String(), bodyReader, header)
	if errors.Is(err, apiutil.ErrDryRun) {
		return cmdutil.PrintDryRun(f.IOStreams, loc, printer, dryRun)
	}
	if err != nil {
		return apiutil.ConvertError(loc, err)
	}
	defer func() { _ = resp.Body.Close() }()

	return printResponse(f, cmd, printer, resp)
}

// parseHeadersは、`--header` の "名前: 値" をヘッダーにする。
// 改行の混入で別のヘッダーを差し込めないよう、名前と値の文字を検査する。
func parseHeaders(f *cmdutil.Factory, cmd *cobra.Command, values []string) (http.Header, error) {
	loc := f.Localizer
	header := http.Header{}

	for i, value := range values {
		name, v, ok := strings.Cut(value, ":")
		// 形式が不正でも資格情報を診断に出さないよう、禁止ヘッダーを先に判定する。
		canonical := http.CanonicalHeaderKey(strings.TrimSpace(name))
		if slices.Contains(forbiddenHeaders, canonical) {
			err := cmdutil.NewUsageError(loc, cmd, loc.T("api_error_forbidden_header", map[string]any{"Name": canonical}))
			err.Hint = loc.T("api_error_forbidden_header_hint")
			return nil, err
		}

		v = strings.TrimSpace(v)
		if !ok || !headerNameRegexp.MatchString(name) || strings.ContainsFunc(v, isForbiddenHeaderValueRune) {
			return nil, cmdutil.NewUsageError(loc, cmd, loc.T("api_error_invalid_header", map[string]any{"Position": i + 1}))
		}

		header.Add(canonical, v)
	}

	return header, nil
}

// isForbiddenHeaderValueRuneは、ヘッダーの値に使えない文字 (タブ以外の制御文字) かを返す。
func isForbiddenHeaderValueRune(r rune) bool {
	return r != '\t' && unicode.IsControl(r)
}

// defaultContentTypeは、`--header` でContent-Typeを指定しないときの本文の種類を返す。
// 公開APIのPATCHはJSON Merge Patchだけを受け付け、それ以外の本文はJSONのため。
func defaultContentType(method string) string {
	if method == http.MethodPatch {
		return "application/merge-patch+json"
	}

	return "application/json"
}

// readInputは、`--input` の本文を読む。指定が無ければnilを返す。`-` なら標準入力から読む。
func readInput(f *cmdutil.Factory, cmd *cobra.Command, input string) ([]byte, error) {
	var (
		body []byte
		err  error
	)
	switch input {
	case "":
		return nil, nil
	case "-":
		body, err = io.ReadAll(f.IOStreams.In)
	default:
		body, err = os.ReadFile(input) // #nosec G304 -- 利用者が本文として指定したファイルを読む
	}
	if err != nil {
		return nil, cmdutil.NewUsageError(f.Localizer, cmd, f.Localizer.T("api_error_read_input", map[string]any{
			"Path": input, "Detail": err.Error(),
		}))
	}

	return body, nil
}

// confirmationは、確認が要る操作の確認の内容を返す。
// 承認後のコマンドには、スペースを置き換えたパスと、要求の中身を決めるフラグを入れる。
// `--header` の値は利用者が渡したもので、秘密のヘッダーはparseHeadersで拒否している。
func confirmation(f *cmdutil.Factory, cmd *cobra.Command, opts *options, method string, ref *url.URL, op apigen.Operation) output.Confirmation {
	loc := f.Localizer

	changes := []string{loc.T("api_confirm_change_request", map[string]any{
		"Method": method, "Path": ref.String(), "Operation": op.ID,
	})}
	if op.Irreversible {
		changes = append(changes, loc.T("api_confirm_change_irreversible"))
	}

	command := append([]string{"wikino", "api", method, ref.String()}, cmdutil.TargetFlagArgs(cmd)...)
	for _, h := range opts.headers {
		command = append(command, "--header", h)
	}
	if opts.input != "" {
		command = append(command, "--input", opts.input)
	}
	command = append(command, "--"+cmdutil.FlagYes)

	return output.Confirmation{Changes: changes, Command: command}
}

// printResponseは、成功の応答の本文をそのまま標準出力に出す。
// JSONの応答は、`--jq` があれば適用し、TTYなら読みやすく字下げする。
// TTYでは、応答に含まれうる端末を操作する制御文字を取り除く。
func printResponse(f *cmdutil.Factory, cmd *cobra.Command, printer *output.Printer, resp *http.Response) error {
	loc, ios := f.Localizer, f.IOStreams
	contentType := resp.Header.Get("Content-Type")
	isJSON := strings.Contains(contentType, "json")
	useJQ := cmd.Flags().Changed(output.FlagJQ)

	// 手を加えずに出せる応答は、溜めずにそのまま流す。
	if !useJQ && !ios.IsStdoutTTY() {
		if _, err := io.Copy(ios.Out, resp.Body); err != nil {
			return readError(f, resp, err)
		}
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return readError(f, resp, err)
	}
	if len(body) == 0 {
		return nil
	}

	if useJQ {
		if !isJSON {
			return &output.Error{
				Code:     output.CodeJQFailed,
				Message:  loc.T("api_error_jq_not_json", map[string]any{"ContentType": contentType}),
				Hint:     loc.T("error_jq_failed_hint"),
				ExitCode: output.ExitError,
			}
		}
		return printer.PrintItem(json.RawMessage(body))
	}

	if !isJSON {
		return output.WriteText(ios, string(body))
	}

	var buf bytes.Buffer
	if err := json.Indent(&buf, body, "", "  "); err != nil {
		// JSONとして読めない本文は、字下げせずに出す。
		buf.Reset()
		buf.Write(body)
	}
	if !bytes.HasSuffix(buf.Bytes(), []byte("\n")) {
		buf.WriteByte('\n')
	}

	return output.WriteText(ios, buf.String())
}

// readErrorは、成功の応答の本文を読み切れなかったことを表すエラーを返す。要求は送っている。
func readError(f *cmdutil.Factory, resp *http.Response, err error) error {
	cliErr := cmdutil.UnexpectedResponseError(f.Localizer, resp.StatusCode)
	cliErr.Err = err

	return cliErr
}

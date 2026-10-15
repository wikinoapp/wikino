package cmdutil

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/apiclient"
	"github.com/wikinoapp/wikino/cli/internal/apiutil"
	"github.com/wikinoapp/wikino/cli/internal/config"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/iostreams"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// APIClientは、cmdのフラグ・環境変数・設定から接続先・スペース・資格情報を決め、
// そのスペースのAPIを呼ぶクライアントとスペースの識別子を返す。
// スペース配下のAPIを呼ぶコマンドは、これでクライアントを得る。
func (f *Factory) APIClient(cmd *cobra.Command) (*apiclient.Client, string, error) {
	return f.APIClientWithDryRun(cmd, nil)
}

// APIClientWithDryRunは、APIClientと同じくクライアントを返す。dryRunがnilでなければ、
// 要求を送る代わりにdryRunへ記録するクライアントにする (`--dry-run`)。
func (f *Factory) APIClientWithDryRun(cmd *cobra.Command, dryRun *apiutil.DryRun) (*apiclient.Client, string, error) {
	dir, err := f.ConfigDir()
	if err != nil {
		return nil, "", err
	}

	cfg, err := config.Load(dir)
	if err != nil {
		return nil, "", config.ConvertError(f.Localizer, err)
	}

	target, err := f.Resolver(cmd, cfg).Resolve()
	if err != nil {
		return nil, "", err
	}

	token, err := f.TokenStore(dir).TokenSource(target)
	if err != nil {
		return nil, "", err
	}

	client, err := f.newAPIClient(target.Host, token, dryRun)
	if err != nil {
		return nil, "", err
	}

	return client, target.Space, nil
}

// FlagDryRunは、要求を送らずに送る要求を表示するフラグの名前。
const FlagDryRun = "dry-run"

// AddDryRunFlagは、`--dry-run` をcmdに足す。書き込みの要求を送るコマンドにだけ足し、
// 読み取りのコマンドに付けて黙って無視されることを避ける。
func AddDryRunFlag(cmd *cobra.Command, loc *i18n.Localizer) *bool {
	return cmd.Flags().Bool(FlagDryRun, false, loc.T("help_dry_run_flag"))
}

// ValidateDryRunFlagは、`--dry-run` が有効なときに呼び、`--fields` との併用を拒否する。
// ページなどの結果用の項目で絞ると、送信予定の要求とdry_runが出力から消えるため、要求前に検査する。
func ValidateDryRunFlag(loc *i18n.Localizer, cmd *cobra.Command) error {
	if cmd.Flags().Changed(output.FlagFields) {
		err := NewUsageError(loc, cmd, loc.T("usage_error_flags_exclusive", map[string]any{"Flags": "--" + FlagDryRun + ", --" + output.FlagFields}))
		err.Hint = loc.T("usage_error_dry_run_fields_hint")
		return err
	}

	return nil
}

// PrintDryRunは、`--dry-run` で送らなかった要求を標準出力にJSONで出し、
// 要求を送っていないことを標準エラーに出す。
func PrintDryRun(ios *iostreams.IOStreams, loc *i18n.Localizer, printer *output.Printer, dryRun *apiutil.DryRun) error {
	if err := printer.PrintItem(dryRun.Output()); err != nil {
		return err
	}

	_, err := fmt.Fprintln(ios.ErrOut, loc.T("dry_run_not_sent"))
	return err
}

// ParseNumberは、位置引数のページ番号・トピック番号 (1以上の整数) を読む。
// 受け付けられない値は、要求を送っていないことを示す使い方の誤りにする。
func ParseNumber(loc *i18n.Localizer, name, value string) (int32, error) {
	n, err := apiclient.ParseNumber(name, value)
	if err != nil {
		return 0, apiutil.ConvertError(loc, err)
	}

	return n, nil
}

// ParseUUIDは、位置引数のID (リビジョンのID・添付ファイルのID) を、ハイフン区切りのUUIDとして読む。
// 受け付けられない値は、要求を送っていないことを示す使い方の誤りにする。
func ParseUUID(loc *i18n.Localizer, name, value string) (uuid.UUID, error) {
	id, err := apiclient.ParseUUID(name, value)
	if err != nil {
		return uuid.UUID{}, apiutil.ConvertError(loc, err)
	}

	return id, nil
}

// ConvertNotFoundは、APIのエラーをCLIのエラーにし、404ならヒントをhintに差し替える。
// 番号を指定して1件を取得するコマンドで、番号を確かめるコマンドを案内するために使う。
func ConvertNotFound(loc *i18n.Localizer, err error, hint string) error {
	return ConvertWithHints(loc, err, map[string]string{output.CodeNotFound: hint})
}

// ConvertWithHintsは、APIのエラーをCLIのエラーにし、エラーのコード (`code`) がhintsにあれば
// ヒントをその値に差し替える。412で取り直すコマンドなど、コマンドに固有の次の手を案内するために使う。
func ConvertWithHints(loc *i18n.Localizer, err error, hints map[string]string) error {
	err = apiutil.ConvertError(loc, err)

	var cliErr *output.Error
	if errors.As(err, &cliErr) {
		if hint, ok := hints[cliErr.Code]; ok {
			cliErr.Hint = hint
		}
	}

	return err
}

// UnexpectedResponseErrorは、成功のステータスでも期待した応答の本文が無いことを表すエラーを返す。
// 要求は送っているため、書き込みのコマンドでは結果を読み取りで確かめるようヒントで案内する。
func UnexpectedResponseError(loc *i18n.Localizer, status int) *output.Error {
	return &output.Error{
		Code:     output.CodeResponse,
		Message:  loc.T("error_response"),
		Hint:     loc.T("error_response_hint"),
		Status:   status,
		ExitCode: output.ExitError,
	}
}

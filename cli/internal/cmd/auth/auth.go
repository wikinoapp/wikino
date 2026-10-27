// Package authは、ログインと資格情報を扱う `wikino auth` のコマンドを提供する。
package auth

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/apigen"
	"github.com/wikinoapp/wikino/cli/internal/auth"
	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/config"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// storeEnvは、WIKINO_TOKENで渡したトークンを使っていることを表すtoken_storeの値。
const storeEnv = "env"

// NewCmdAuthは `wikino auth` を返す。
func NewCmdAuth(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	cmd := &cobra.Command{
		Use:     "auth",
		Short:   loc.T("auth_short"),
		Long:    loc.T("auth_long"),
		Example: loc.T("auth_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("auth_output"),
			cmdutil.AnnotationErrors: loc.T("auth_errors"),
		},
	}
	cmdutil.ConfigureGroup(cmd, loc)

	cmd.AddCommand(newCmdLogin(f))
	cmd.AddCommand(newCmdLogout(f))
	cmd.AddCommand(newCmdStatus(f))
	cmd.AddCommand(newCmdSwitch(f))

	return cmd
}

// credentialFieldsは、資格情報の組を出すコマンド (login・logout・switch) の `--fields` で選べる項目。
var credentialFields = []string{"host", "space", "member", "token_store", "default", "space_default"}

// credentialResultは、資格情報の組の出力。
type credentialResult struct {
	Host         string       `json:"host"`
	Space        string       `json:"space"`
	Member       memberResult `json:"member"`
	TokenStore   string       `json:"token_store"`
	Default      bool         `json:"default"`
	SpaceDefault bool         `json:"space_default"`
	// ScopesはOAuthでログインしたときに付与されたスコープ。`auth login` でだけ出す。
	Scopes []string `json:"scopes,omitempty"`
}

// memberResultはスペースメンバーの出力。表示名はAPIから取得したときだけ出す。
type memberResult struct {
	ID     string  `json:"id"`
	Atname string  `json:"atname"`
	Name   *string `json:"name,omitempty"`
}

func newCredentialResult(cfg *config.Config, cred *config.Credential, member *apigen.SpaceMember) credentialResult {
	result := credentialResult{
		Host:         cred.Host,
		Space:        cred.Space,
		Member:       memberResult{ID: cred.MemberID, Atname: cred.MemberAtname},
		TokenStore:   cred.Store,
		Default:      cfg.IsDefault(cred),
		SpaceDefault: cred.SpaceDefault,
	}
	if member != nil {
		result.Member.Name = &member.Name
	}

	return result
}

// fieldsは、人間向けの表示で並べる項目。
func (r credentialResult) fields(loc *i18n.Localizer) []output.Field {
	fields := []output.Field{
		{Label: loc.T("auth_label_host"), Value: r.Host},
		{Label: loc.T("auth_label_space"), Value: r.Space},
		{Label: loc.T("auth_label_member"), Value: r.Member.display()},
		{Label: loc.T("auth_label_token_store"), Value: tokenStoreLabel(loc, r.TokenStore)},
		{Label: loc.T("auth_label_default"), Value: defaultLabel(loc, r.Default, r.SpaceDefault)},
	}
	if len(r.Scopes) > 0 {
		fields = append(fields, output.Field{Label: loc.T("auth_label_scopes"), Value: strings.Join(r.Scopes, " ")})
	}

	return fields
}

func (m memberResult) display() string {
	if m.Name == nil || *m.Name == "" {
		return "@" + m.Atname
	}

	return "@" + m.Atname + " (" + *m.Name + ")"
}

func tokenStoreLabel(loc *i18n.Localizer, store string) string {
	switch store {
	case auth.StoreKeyring:
		return loc.T("auth_value_store_keyring")
	case auth.StoreFile:
		return loc.T("auth_value_store_file")
	case storeEnv:
		return loc.T("auth_value_store_env")
	default:
		return store
	}
}

func defaultLabel(loc *i18n.Localizer, isDefault, spaceDefault bool) string {
	switch {
	case isDefault:
		return loc.T("auth_value_default")
	case spaceDefault:
		return loc.T("auth_value_space_default")
	default:
		return loc.T("auth_value_not_default")
	}
}

// printCredentialは資格情報の組を出す。
func printCredential(printer *output.Printer, loc *i18n.Localizer, result credentialResult) error {
	if printer.IsJSON() {
		return printer.PrintItem(result)
	}

	return printer.PrintDetail(result.fields(loc), "")
}

// spaceArgは、位置引数のスペースを返す。位置引数が無ければ `--space` の値を使う。
// 両方を違う値で指定したときは、どちらを使うかが曖昧なため使い方の誤りにする。
func spaceArg(f *cmdutil.Factory, cmd *cobra.Command, args []string) (string, error) {
	flag := config.ReadFlags(cmd.Flags()).Space
	if len(args) == 0 {
		return flag, nil
	}
	if flag != "" && !strings.EqualFold(flag, args[0]) {
		return "", cmdutil.NewUsageError(f.Localizer, cmd, f.Localizer.T("usage_error_space_conflict"))
	}

	return args[0], nil
}

// loadConfigは設定のディレクトリと設定を読む。
func loadConfig(f *cmdutil.Factory) (string, *config.Config, error) {
	dir, err := f.ConfigDir()
	if err != nil {
		return "", nil, err
	}

	cfg, err := config.Load(dir)
	if err != nil {
		return "", nil, config.ConvertError(f.Localizer, err)
	}

	return dir, cfg, nil
}

// withLockedConfigは、config.WithLockedで最新の設定を更新する。設定ファイルを解析できないときは
// 翻訳したエラーにする。
func withLockedConfig(ctx context.Context, f *cmdutil.Factory, dir string, fn func(*config.Config) error) error {
	return config.ConvertError(f.Localizer, config.WithLocked(ctx, dir, fn))
}

// revokeRefreshTokenは、リフレッシュトークンを認可サーバーで失効させ、失効を確かめられたかを返す。
// ログアウトと、ログインし直して置き換えたトークンの失効で使う。
func revokeRefreshToken(f *cmdutil.Factory, ctx context.Context, host, refreshToken string) bool {
	client := f.OAuthClient(host)
	metadata, err := client.FetchMetadata(ctx)
	if err == nil {
		err = client.Revoke(ctx, metadata, refreshToken)
	}

	return err == nil
}

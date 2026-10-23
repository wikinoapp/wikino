package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/apiclient"
	"github.com/wikinoapp/wikino/cli/internal/apigen"
	"github.com/wikinoapp/wikino/cli/internal/apiutil"
	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/config"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// statusFieldsは `auth status` の `--fields` で選べる項目。
var statusFields = append(append([]string{}, credentialFields...), "token", "error")

// statusResultは、`auth status` の1件の出力。
type statusResult struct {
	credentialResult
	// TokenはAPIから取得したトークンの情報。取得できなかったときはnil。
	Token *tokenResult `json:"token"`
	// Errorは、トークンの情報を取得できなかった理由。
	Error *statusError `json:"error,omitempty"`
}

type tokenResult struct {
	Kind      string    `json:"kind"`
	Scopes    []string  `json:"scopes"`
	ExpiresAt time.Time `json:"expires_at"`
}

type statusError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
	Status  int    `json:"status,omitempty"`

	exitCode int
}

type statusList struct {
	Items []statusResult `json:"items"`
}

func newCmdStatus(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer

	return &cobra.Command{
		Use:     "status",
		Short:   loc.T("auth_status_short"),
		Long:    loc.T("auth_status_long"),
		Example: loc.T("auth_status_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("auth_status_output"),
			cmdutil.AnnotationErrors: loc.T("auth_status_errors"),
		},
		Args: cmdutil.ExactArgs(loc, 0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runStatus(f, cmd)
		},
	}
}

func runStatus(f *cmdutil.Factory, cmd *cobra.Command) error {
	loc := f.Localizer

	printer, err := output.NewPrinter(f.IOStreams, loc, cmd.Flags(), output.Spec{Fields: statusFields})
	if err != nil {
		return err
	}

	dir, cfg, err := loadConfig(f)
	if err != nil {
		return err
	}
	resolver := f.Resolver(cmd, cfg)

	var items []statusResult
	if f.Getenv(config.EnvToken) != "" {
		target, err := resolver.Resolve()
		if err != nil {
			return err
		}
		items = append(items, checkEnvToken(f, cmd.Context(), target))
	} else {
		creds, err := selectCredentials(resolver, cfg)
		if err != nil {
			return err
		}

		var updates []atnameUpdate
		for _, cred := range creds {
			item, member := checkCredential(f, cmd.Context(), dir, cfg, cred)
			if member != nil && member.Atname != cred.MemberAtname {
				updates = append(updates, atnameUpdate{credential: *cred, atname: member.Atname})
			}
			items = append(items, item)
		}
		if len(updates) > 0 {
			if err := updateAtnames(cmd.Context(), dir, updates); err != nil {
				return config.ConvertError(f.Localizer, err)
			}
		}

	}

	if err := printStatus(printer, f, items); err != nil {
		return err
	}

	// 確かめられなかった資格情報があれば、成功と取り違えないよう最初の失敗の終了コードで終える。
	// 理由は各項目の `error` に出しているため、標準エラーには重ねて出さない。
	for _, item := range items {
		if item.Error != nil {
			return &output.SilentError{ExitCode: item.Error.exitCode}
		}
	}

	return nil
}

// atnameUpdateは、APIを呼ぶ前の資格情報と取得したアットネームの組。
type atnameUpdate struct {
	credential config.Credential
	atname     string
}

// updateAtnamesは最新の設定へ名前だけを反映し、並行したログインやログアウトを上書きしない。
// APIの取得中に名前が更新された場合は、その後の変更を優先する。
func updateAtnames(ctx context.Context, dir string, updates []atnameUpdate) error {
	return config.WithLocked(ctx, dir, func(cfg *config.Config) error {
		changed := false
		for _, update := range updates {
			old := update.credential
			for _, cred := range cfg.FindCredentials(old.Host, old.Space) {
				if cred.MemberID == old.MemberID && cred.MemberAtname == old.MemberAtname && cred.Store == old.Store {
					cred.MemberAtname = update.atname
					changed = true
				}
			}
		}
		if !changed {
			return nil
		}
		return cfg.Save(dir)
	})
}

// selectCredentialsは、表示する資格情報を返す。`--host`・`--space`・`--member` を指定したときは、
// 一致するものだけに絞る。
func selectCredentials(resolver *config.Resolver, cfg *config.Config) ([]*config.Credential, error) {
	flags := resolver.Flags
	host := ""
	if flags.Host != "" {
		var err error
		if host, err = resolver.ResolveHost(); err != nil {
			return nil, err
		}
	}

	var selected []*config.Credential
	for i := range cfg.Credentials {
		cred := &cfg.Credentials[i]
		if (host == "" || cred.Host == host) &&
			(flags.Space == "" || strings.EqualFold(cred.Space, flags.Space)) &&
			(flags.Member == "" || strings.EqualFold(cred.MemberAtname, flags.Member)) {
			selected = append(selected, cred)
		}
	}

	if len(selected) == 0 {
		if host == "" {
			host = config.DefaultHost
			if cfg.Default.Host != "" {
				host = cfg.Default.Host
			}
		}
		return nil, resolver.NotLoggedInError(host, "")
	}

	return selected, nil
}

func checkCredential(f *cmdutil.Factory, ctx context.Context, dir string, cfg *config.Config, cred *config.Credential) (statusResult, *apigen.SpaceMember) {
	item := statusResult{credentialResult: newCredentialResult(cfg, cred, nil)}

	token, err := f.TokenStore(dir).TokenSource(&config.Target{Host: cred.Host, Space: cred.Space, Credential: cred})
	if err != nil {
		item.Error = toStatusError(f, err)
		return item, nil
	}

	member := fetchToken(f, ctx, cred.Host, cred.Space, token, &item)
	if member != nil {
		item.Member.Atname = member.Atname
	}

	return item, member
}

func checkEnvToken(f *cmdutil.Factory, ctx context.Context, target *config.Target) statusResult {
	item := statusResult{credentialResult: credentialResult{Host: target.Host, Space: target.Space, TokenStore: storeEnv}}

	member := fetchToken(f, ctx, target.Host, target.Space, apiclient.StaticToken(target.EnvToken), &item)
	if member != nil {
		item.Member.ID, item.Member.Atname = member.Id.String(), member.Atname
	}

	return item
}

// fetchTokenは、トークンの持ち主のメンバーとトークンの情報をAPIから取得してitemに入れる。
// 取得できなかったときはitemのErrorに理由を入れ、nilを返す。
func fetchToken(f *cmdutil.Factory, ctx context.Context, host, space string, token apiclient.TokenSource, item *statusResult) *apigen.SpaceMember {
	client, err := f.NewAPIClient(host, token)
	if err != nil {
		item.Error = toStatusError(f, err)
		return nil
	}

	memberResp, err := client.GetCurrentSpaceMemberWithResponse(ctx, space)
	if err != nil {
		item.Error = toStatusError(f, apiutil.ConvertError(f.Localizer, err))
		return nil
	}
	member := memberResp.JSON200
	if member == nil {
		item.Error = toStatusError(f, unexpectedResponseError(f, memberResp.StatusCode()))
		return nil
	}
	item.Member.Name = &member.Name

	tokenResp, err := client.GetCurrentTokenWithResponse(ctx, space)
	if err != nil {
		item.Error = toStatusError(f, apiutil.ConvertError(f.Localizer, err))
		return member
	}
	if tokenResp.JSON200 == nil {
		item.Error = toStatusError(f, unexpectedResponseError(f, tokenResp.StatusCode()))
		return member
	}
	item.Token = &tokenResult{
		Kind:      string(tokenResp.JSON200.Kind),
		Scopes:    tokenResp.JSON200.Scopes,
		ExpiresAt: tokenResp.JSON200.ExpiresAt,
	}

	return member
}

func toStatusError(f *cmdutil.Factory, err error) *statusError {
	var cliErr *output.Error
	if !errors.As(err, &cliErr) {
		cliErr = &output.Error{Code: output.CodeUnexpected, Message: err.Error(), ExitCode: output.ExitError}
	}

	return &statusError{Code: cliErr.Code, Message: cliErr.Message, Hint: cliErr.Hint, Status: cliErr.Status, exitCode: cliErr.ExitCode}
}

func printStatus(printer *output.Printer, f *cmdutil.Factory, items []statusResult) error {
	if printer.IsJSON() {
		return printer.PrintList(statusList{Items: items})
	}

	loc := f.Localizer
	for i, item := range items {
		if i > 0 {
			if _, err := f.IOStreams.Out.Write([]byte("\n")); err != nil {
				return err
			}
		}

		fields := item.fields(loc)
		if item.Token != nil {
			fields = append(fields,
				output.Field{Label: loc.T("auth_label_token_kind"), Value: item.Token.Kind},
				output.Field{Label: loc.T("auth_label_scopes"), Value: strings.Join(item.Token.Scopes, " ")},
				output.Field{Label: loc.T("auth_label_expires_at"), Value: item.Token.ExpiresAt.Format(time.RFC3339)},
			)
		}
		if item.Error != nil {
			fields = append(fields, output.Field{Label: loc.T("auth_label_error"), Value: item.Error.Message + " " + item.Error.Hint})
		}

		if err := printer.PrintDetail(fields, ""); err != nil {
			return err
		}
	}

	return nil
}

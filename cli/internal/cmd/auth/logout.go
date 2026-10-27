package auth

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/auth"
	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/config"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

func newCmdLogout(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer

	return &cobra.Command{
		Use:     "logout [<space>]",
		Short:   loc.T("auth_logout_short"),
		Long:    loc.T("auth_logout_long"),
		Example: loc.T("auth_logout_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("auth_logout_output"),
			cmdutil.AnnotationErrors: loc.T("auth_logout_errors"),
		},
		Args: cmdutil.MaximumNArgs(loc, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogout(f, cmd, args)
		},
	}
}

func runLogout(f *cmdutil.Factory, cmd *cobra.Command, args []string) error {
	loc := f.Localizer

	space, err := spaceArg(f, cmd, args)
	if err != nil {
		return err
	}

	printer, err := output.NewPrinter(f.IOStreams, loc, cmd.Flags(), output.Spec{Fields: credentialFields})
	if err != nil {
		return err
	}

	dir, err := f.ConfigDir()
	if err != nil {
		return err
	}
	var result credentialResult
	if err := withLockedConfig(cmd.Context(), f, dir, func(cfg *config.Config) error {
		resolver := f.Resolver(cmd, cfg)
		resolver.Flags.Space = space
		// WIKINO_TOKENは保存していないため、対象は保存した資格情報だけにする。
		target, err := resolver.ResolveCredential()
		if err != nil {
			return err
		}
		cred := *target.Credential
		result = newCredentialResult(cfg, &cred, nil)
		// 同じ資格情報のリフレッシュが、消した後のトークンを書き戻さないようにする。
		return auth.WithCredentialLock(cmd.Context(), dir, &cred, func() error {
			store := f.TokenStore(dir)
			revoked := revokeToken(f, cmd.Context(), store, &cred)
			// 削除に失敗した場合は設定を残し、旧保存先も含めて再試行できるようにする。
			// PATはサーバーで失効させないため、無効にするにはスペース設定で削除する。
			if err := store.Delete(&cred); err != nil {
				return authStorageError(f, "auth_error_cleanup", err)
			}
			cfg.RemoveCredential(&cred)
			if err := cfg.Save(dir); err != nil {
				return err
			}
			// 警告は手元の資格情報を消したことも伝えるため、消し終えてから出す。
			if !revoked {
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, f.Localizer.T("auth_warning_revoke_failed", map[string]any{"Space": cred.Space}))
			}
			return nil
		})
	}); err != nil {
		return err
	}

	return printCredential(printer, loc, result)
}

// revokeTokenは、OAuthで得たトークンなら、リフレッシュトークンを認可サーバーで失効させる。
// 失効を確かめられなかったとき (トークンを読めない・接続先に届かないなど) はfalseを返す。
// 呼び出し側は失効できなくても手元の資格情報を消し、連携中のアプリの画面で取り消せることを警告する。
// 手元に残すと、使えない接続先の資格情報を消せなくなるため。
// PATはサーバーで失効させないため、失効を待たずにtrueを返す。
func revokeToken(f *cmdutil.Factory, ctx context.Context, store *auth.Store, cred *config.Credential) bool {
	secret, err := store.Load(cred)
	if err != nil {
		return false
	}
	if !secret.IsOAuth() {
		return true
	}

	return revokeRefreshToken(f, ctx, cred.Host, secret.RefreshToken)
}

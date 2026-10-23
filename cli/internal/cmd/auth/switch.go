package auth

import (
	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/config"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

func newCmdSwitch(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer

	return &cobra.Command{
		Use:     "switch <space>",
		Short:   loc.T("auth_switch_short"),
		Long:    loc.T("auth_switch_long"),
		Example: loc.T("auth_switch_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("auth_switch_output"),
			cmdutil.AnnotationErrors: loc.T("auth_switch_errors"),
		},
		Args: cmdutil.ExactArgs(loc, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSwitch(f, cmd, args)
		},
	}
}

func runSwitch(f *cmdutil.Factory, cmd *cobra.Command, args []string) error {
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
		host, err := resolver.ResolveHost()
		if err != nil {
			return err
		}
		if err := resolver.ValidateSpace("<space>", space); err != nil {
			return err
		}
		// 既定を書き換えるため、メンバーは環境変数ではなく明示されたフラグだけで選ぶ。
		cred, err := resolver.ResolveMember(host, space, resolver.Flags.Member)
		if err != nil {
			return err
		}
		cfg.SetDefault(cred)
		if err := cfg.Save(dir); err != nil {
			return err
		}
		result = newCredentialResult(cfg, cred, nil)
		return nil
	}); err != nil {
		return err
	}
	return printCredential(printer, loc, result)

}

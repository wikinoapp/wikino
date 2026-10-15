// Package skillは、コーディングエージェント向けのスキルを扱う `wikino skill` のコマンドを提供する。
package skill

import (
	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
)

// NewCmdSkillは `wikino skill` を返す。
func NewCmdSkill(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	cmd := &cobra.Command{
		Use:     "skill",
		Short:   loc.T("skill_short"),
		Long:    loc.T("skill_long"),
		Example: loc.T("skill_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("skill_output"),
			cmdutil.AnnotationErrors: loc.T("skill_errors"),
		},
	}
	cmdutil.ConfigureGroup(cmd, loc)

	cmd.AddCommand(newCmdInstall(f))

	return cmd
}

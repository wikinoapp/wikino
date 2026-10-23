// Package attachmentは、添付ファイルを扱う `wikino attachment` のコマンドを提供する。
package attachment

import (
	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
)

// NewCmdAttachmentは `wikino attachment` を返す。
func NewCmdAttachment(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	cmd := &cobra.Command{
		Use:     "attachment",
		Short:   loc.T("attachment_short"),
		Long:    loc.T("attachment_long"),
		Example: loc.T("attachment_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("attachment_output"),
			cmdutil.AnnotationErrors: loc.T("attachment_errors"),
		},
	}
	cmdutil.ConfigureGroup(cmd, loc)

	cmd.AddCommand(newCmdGet(f))

	return cmd
}

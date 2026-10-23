package draft

import (
	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
)

// revisionFieldsは、APIの下書きのリビジョンの項目。リビジョンは作成した後に変わらないため、版 (etag) を持たない。
var revisionFields = []string{"id", "title", "created_at", "body"}

// newCmdRevisionは `wikino draft revision` を返す。下書きの編集履歴 (リビジョン) を扱うサブコマンドを束ねる。
func newCmdRevision(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	cmd := &cobra.Command{
		Use:     "revision",
		Short:   loc.T("draft_revision_short"),
		Long:    loc.T("draft_revision_long"),
		Example: loc.T("draft_revision_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("draft_revision_output"),
			cmdutil.AnnotationErrors: loc.T("draft_revision_errors"),
		},
	}
	cmdutil.ConfigureGroup(cmd, loc)

	cmd.AddCommand(newCmdRevisionList(f))
	cmd.AddCommand(newCmdRevisionGet(f))

	return cmd
}

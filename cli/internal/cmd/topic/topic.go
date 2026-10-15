// Package topicは、トピックを扱う `wikino topic` のコマンドを提供する。
package topic

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/apigen"
	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
)

// NewCmdTopicは `wikino topic` を返す。
func NewCmdTopic(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	cmd := &cobra.Command{
		Use:     "topic",
		Short:   loc.T("topic_short"),
		Long:    loc.T("topic_long"),
		Example: loc.T("topic_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("topic_output"),
			cmdutil.AnnotationErrors: loc.T("topic_errors"),
		},
	}
	cmdutil.ConfigureGroup(cmd, loc)

	cmd.AddCommand(newCmdList(f))
	cmd.AddCommand(newCmdGet(f))

	return cmd
}

// topicFieldsは、トピックを出すコマンドの `--fields` で選べる項目。APIのトピックの項目と同じ。
var topicFields = []string{"id", "number", "name", "description", "visibility"}

// visibilityLabelは、人間向けの表示で使う公開範囲の名前を返す。
func visibilityLabel(loc *i18n.Localizer, visibility apigen.TopicVisibility) string {
	switch visibility {
	case apigen.Public:
		return loc.T("topic_value_public")
	case apigen.Private:
		return loc.T("topic_value_private")
	default:
		return string(visibility)
	}
}

func formatNumber(n int32) string {
	return strconv.FormatInt(int64(n), 10)
}

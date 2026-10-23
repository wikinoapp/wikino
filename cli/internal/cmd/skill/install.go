package skill

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
	skillfiles "github.com/wikinoapp/wikino/cli/internal/skill"
)

const flagDir = "dir"

// claudeConfigDirEnvは、Claude Codeが設定のディレクトリを変えるときに使う環境変数。
// 既定の書き出し先を、Claude Codeが実際に読むユーザー単位のスキルのディレクトリに合わせるために読む。
const claudeConfigDirEnv = "CLAUDE_CONFIG_DIR"

// ディレクトリの権限はumaskに従う。スキルのファイルは秘密の値を含まないため、0644にする。
const (
	dirPerm  = 0o755
	filePerm = 0o644
)

// installFieldsは、`skill install` の `--fields` で選べる項目。
var installFields = []string{"path", "lang", "files"}

// installItemは、`skill install` の出力。
type installItem struct {
	// Pathは書き出したスキルのディレクトリ。
	Path string `json:"path"`
	// Langは書き出したスキルの言語。
	Lang string `json:"lang"`
	// Filesは書き出したファイルの名前。
	Files []string `json:"files"`
}

type installOptions struct {
	dir string
}

func newCmdInstall(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	opts := &installOptions{}
	cmd := &cobra.Command{
		Use:     "install",
		Short:   loc.T("skill_install_short"),
		Long:    loc.T("skill_install_long"),
		Example: loc.T("skill_install_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("skill_install_output"),
			cmdutil.AnnotationErrors: loc.T("skill_install_errors"),
		},
		Args: cmdutil.ExactArgs(loc, 0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInstall(f, cmd, opts)
		},
	}
	cmd.Flags().StringVar(&opts.dir, flagDir, "", loc.T("skill_install_dir_flag"))

	return cmd
}

func runInstall(f *cmdutil.Factory, cmd *cobra.Command, opts *installOptions) error {
	loc := f.Localizer

	printer, err := output.NewPrinter(f.IOStreams, loc, cmd.Flags(), output.Spec{Fields: installFields})
	if err != nil {
		return err
	}

	skillsDir := opts.dir
	if skillsDir == "" {
		if skillsDir, err = defaultSkillsDir(f.Getenv); err != nil {
			return &output.Error{
				Code:     output.CodeUsage,
				Message:  loc.T("skill_install_error_no_home", map[string]any{"Detail": err.Error()}),
				Hint:     loc.T("skill_install_error_no_home_hint"),
				ExitCode: output.ExitUsage,
				Err:      err,
			}
		}
	}

	files, err := skillfiles.Files(loc.Lang())
	if err != nil {
		return err
	}

	// 書き出し先の名前をスキルの名前に固定し、指定したディレクトリにある他のスキルのファイルを上書きしない。
	dir := filepath.Join(skillsDir, skillfiles.Name)
	item := installItem{Path: dir, Lang: loc.Lang(), Files: make([]string, 0, len(files))}
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return newWriteError(loc, dir, err)
	}
	for _, file := range files {
		path := filepath.Join(dir, file.Name)
		if err := writeFile(path, file.Content); err != nil {
			return newWriteError(loc, path, err)
		}
		item.Files = append(item.Files, file.Name)
	}

	if printer.IsJSON() {
		return printer.PrintItem(item)
	}

	return printer.PrintDetail([]output.Field{
		{Label: loc.T("skill_label_path"), Value: item.Path},
		{Label: loc.T("skill_label_lang"), Value: item.Lang},
		{Label: loc.T("skill_label_files"), Value: strings.Join(item.Files, ", ")},
	}, "")
}

// defaultSkillsDirは、スキルを置くユーザー単位のディレクトリ (`$CLAUDE_CONFIG_DIR/skills`、未設定なら `~/.claude/skills`) を返す。
// リポジトリの `.claude/skills/` (そのリポジトリの開発用のスキル) と混ざらないよう、ユーザー単位の場所にする。
func defaultSkillsDir(getenv func(string) string) (string, error) {
	if dir := getenv(claudeConfigDirEnv); dir != "" {
		return filepath.Join(dir, "skills"), nil
	}

	home := getenv("HOME")
	if home == "" {
		var err error
		if home, err = os.UserHomeDir(); err != nil {
			return "", err
		}
	}

	return filepath.Join(home, ".claude", "skills"), nil
}

// writeFileは、contentをpathに書く。同じディレクトリの一時ファイルに書いてから置き換え、
// 書き込みに失敗しても、前に書き出したスキルのファイルを書きかけの状態にしない。
func writeFile(path string, content []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".wikino-skill-*")
	if err != nil {
		return err
	}
	_, err = file.Write(content)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Chmod(file.Name(), filePerm)
	}
	if err == nil {
		err = os.Rename(file.Name(), path)
	}
	if err != nil {
		_ = os.Remove(file.Name())
	}

	return err
}

// newWriteErrorは、pathに書き込めなかったことを表すエラーを返す。
// 原因がパスの操作のエラーなら、メッセージにパスを重ねて出さないよう、原因だけを説明に使う。
func newWriteError(loc *i18n.Localizer, path string, err error) *output.Error {
	detail := err
	var pathErr *fs.PathError
	var linkErr *os.LinkError
	switch {
	case errors.As(err, &pathErr):
		detail = pathErr.Err
	case errors.As(err, &linkErr):
		detail = linkErr.Err
	}

	return &output.Error{
		Code:     output.CodeWriteFailed,
		Message:  loc.T("skill_install_error_write", map[string]any{"Path": path, "Detail": detail.Error()}),
		Hint:     loc.T("skill_install_error_write_hint"),
		ExitCode: output.ExitError,
		Err:      err,
	}
}

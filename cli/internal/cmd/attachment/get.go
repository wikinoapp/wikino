package attachment

import (
	"crypto/rand"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/apiclient"
	"github.com/wikinoapp/wikino/cli/internal/apiutil"
	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// 保存先を選ぶフラグの名前。
const (
	flagOutput  = "output"
	flagClobber = "clobber"
)

// stdoutOutputは、`--output` で標準出力を指す値。
const stdoutOutput = "-"

// filePermは、保存するファイルの権限。添付ファイルはスペースのメンバーが見られる中身で秘密の値ではないため、
// `curl -o` などと同じくumaskに従わせる。
const filePerm = 0o666

// referencePrefixは、本文が添付ファイルを参照するパスのIDより前の部分。
// 本文から写した `/attachments/{ID}` をそのまま引数に渡せるよう、取り除いてからIDとして読む。
const referencePrefix = "/attachments/"

// getFieldsは、`attachment get` の `--fields` で選べる項目。
var getFields = []string{"id", "path", "filename", "content_type", "byte_size"}

// getItemは、`attachment get` の出力。署名付きURLは取得の権限を持つため出さない。
type getItem struct {
	ID string `json:"id"`
	// Pathは保存したファイルのパス。
	Path string `json:"path"`
	// FilenameはAPIが返した元のファイル名。保存先の名前には整えた名前を使う。
	Filename    string  `json:"filename"`
	ContentType *string `json:"content_type"`
	// ByteSizeは保存したバイト数。
	ByteSize int64 `json:"byte_size"`
}

type getOptions struct {
	output  string
	clobber bool
}

func newCmdGet(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	opts := &getOptions{}
	cmd := &cobra.Command{
		Use:     "get <attachment_id>",
		Short:   loc.T("attachment_get_short"),
		Long:    loc.T("attachment_get_long"),
		Example: loc.T("attachment_get_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("attachment_get_output"),
			cmdutil.AnnotationErrors: loc.T("attachment_get_errors"),
		},
		Args: cmdutil.ExactArgs(loc, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(f, cmd, args[0], opts)
		},
	}
	cmd.Flags().StringVarP(&opts.output, flagOutput, "O", "", loc.T("attachment_get_output_flag"))
	cmd.Flags().BoolVar(&opts.clobber, flagClobber, false, loc.T("attachment_get_clobber_flag"))

	return cmd
}

func runGet(f *cmdutil.Factory, cmd *cobra.Command, arg string, opts *getOptions) error {
	loc := f.Localizer

	id, err := cmdutil.ParseUUID(loc, "<attachment_id>", strings.TrimPrefix(arg, referencePrefix))
	if err != nil {
		return err
	}

	toStdout := opts.output == stdoutOutput
	var printer *output.Printer
	if toStdout {
		if err := validateStdoutFlags(loc, cmd); err != nil {
			return err
		}
		// 添付ファイルの中身は他の利用者が上げたもので、端末を操作するエスケープシーケンスを含みうる。
		// 中身を書き換えると保存したファイルが壊れるため、取り除かずに端末への書き出しを断る。
		if f.IOStreams.IsStdoutTTY() {
			usageErr := cmdutil.NewUsageError(loc, cmd, loc.T("attachment_get_error_stdout_tty"))
			usageErr.Hint = loc.T("attachment_get_error_stdout_tty_hint")
			return usageErr
		}
	} else if printer, err = output.NewPrinter(f.IOStreams, loc, cmd.Flags(), output.Spec{Fields: getFields}); err != nil {
		return err
	}

	// 要求を送る前に、保存先のディレクトリがあることと、保存先がファイルを指していれば既存のファイルが無いことを確かめる。
	dir, path, err := resolveDestination(opts.output)
	if err != nil {
		return newWriteError(loc, opts.output, err)
	}
	if !toStdout {
		if err := checkDirectory(loc, dir, path); err != nil {
			return err
		}
		if path != "" {
			if err := checkNotExists(loc, path, opts.clobber); err != nil {
				return err
			}
		}
	}

	client, space, err := f.APIClient(cmd)
	if err != nil {
		return err
	}

	resp, err := client.GetAttachmentWithResponse(cmd.Context(), space, id)
	if err != nil {
		return cmdutil.ConvertNotFound(loc, err, loc.T("attachment_get_error_not_found_hint"))
	}
	attachment := resp.JSON200
	if attachment == nil {
		return cmdutil.UnexpectedResponseError(loc, resp.StatusCode())
	}

	if !toStdout && path == "" {
		path = filepath.Join(dir, sanitizeFilename(attachment.Filename, attachment.Id.String()))
		if err := checkNotExists(loc, path, opts.clobber); err != nil {
			return err
		}
	}

	download, err := client.Download(cmd.Context(), attachment.DownloadUrl)
	if err != nil {
		return apiutil.ConvertError(loc, err)
	}
	defer func() { _ = download.Body.Close() }()

	if toStdout {
		if _, err := io.Copy(f.IOStreams.Out, download.Body); err != nil {
			return convertSaveError(loc, err, func(err error) error {
				return newWriteFailedError(loc.T("attachment_get_error_write_stdout", map[string]any{"Detail": err.Error()}), "", err)
			})
		}
		return nil
	}

	size, err := saveFile(path, download.Body, opts.clobber)
	// clobberの置き換えは、保存先がディレクトリのときにもErrExistで失敗するため、書き込みの失敗として扱う。
	if !opts.clobber && errors.Is(err, fs.ErrExist) {
		return newFileExistsError(loc, path)
	}
	if err != nil {
		return convertSaveError(loc, err, func(err error) error {
			return newWriteError(loc, path, err)
		})
	}

	item := getItem{
		ID:          attachment.Id.String(),
		Path:        path,
		Filename:    attachment.Filename,
		ContentType: attachment.ContentType,
		ByteSize:    size,
	}
	if printer.IsJSON() {
		return printer.PrintItem(item)
	}

	contentType := loc.T("attachment_value_unknown_content_type")
	if item.ContentType != nil {
		contentType = *item.ContentType
	}

	return printer.PrintDetail([]output.Field{
		{Label: loc.T("attachment_label_path"), Value: item.Path},
		{Label: loc.T("attachment_label_filename"), Value: item.Filename},
		{Label: loc.T("attachment_label_content_type"), Value: contentType},
		{Label: loc.T("attachment_label_byte_size"), Value: strconv.FormatInt(item.ByteSize, 10)},
	}, "")
}

// validateStdoutFlagsは、標準出力に中身を出すときに意味を持たないフラグが指定されていないかを検査する。
// 中身はJSONでなく、既存のファイルも上書きしないため、黙って無視されるのを避ける。
func validateStdoutFlags(loc *i18n.Localizer, cmd *cobra.Command) error {
	for _, name := range []string{output.FlagJSON, output.FlagFields, output.FlagJQ, flagClobber} {
		if cmd.Flags().Changed(name) {
			return cmdutil.NewUsageError(loc, cmd, loc.T("usage_error_flags_exclusive", map[string]any{
				"Flags": "--" + flagOutput + " " + stdoutOutput + ", --" + name,
			}))
		}
	}

	return nil
}

// resolveDestinationは、`--output` の値から保存先を決める。
// 保存先のファイルが決まればpathを、元のファイル名で保存するディレクトリならdirを返す。
// 省略時と、既存のディレクトリ・パスの区切りで終わる値はディレクトリとして扱う。
func resolveDestination(value string) (dir, path string, err error) {
	switch {
	case value == "":
		return ".", "", nil
	case value == stdoutOutput:
		return "", "", nil
	case strings.HasSuffix(value, "/") || strings.HasSuffix(value, string(filepath.Separator)):
		return value, "", nil
	}

	info, err := os.Stat(value)
	switch {
	case err == nil && info.IsDir():
		return value, "", nil
	// 親がディレクトリでない (ENOTDIR) 値も、無いファイルとして扱い、親の確認 (checkDirectory) で使い方の誤りにする。
	case err == nil || errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR):
		return "", value, nil
	default:
		return "", "", err
	}
}

// checkDirectoryは、保存先のディレクトリ (dir、またはファイルのパスpathの親) があることを確かめる。
// 無ければ、中身を取得してから失敗しないよう、要求の前に使い方の誤りにする。
func checkDirectory(loc *i18n.Localizer, dir, path string) error {
	if path != "" {
		dir = filepath.Dir(path)
	}

	info, err := os.Stat(dir)
	switch {
	case err == nil && info.IsDir():
		return nil
	case err == nil || errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR):
		return &output.Error{
			Code:     output.CodeUsage,
			Message:  loc.T("attachment_get_error_no_directory", map[string]any{"Path": dir}),
			Hint:     loc.T("attachment_get_error_no_directory_hint"),
			ExitCode: output.ExitUsage,
		}
	default:
		return newWriteError(loc, dir, err)
	}
}

// checkNotExistsは、clobberでなければ、pathに既存のファイル (壊れたシンボリックリンクを含む) が無いことを確かめる。
func checkNotExists(loc *i18n.Localizer, path string, clobber bool) error {
	if clobber {
		return nil
	}

	_, err := os.Lstat(path)
	switch {
	case err == nil:
		return newFileExistsError(loc, path)
	case errors.Is(err, fs.ErrNotExist):
		return nil
	default:
		return newWriteError(loc, path, err)
	}
}

func newFileExistsError(loc *i18n.Localizer, path string) *output.Error {
	return &output.Error{
		Code:     output.CodeFileExists,
		Message:  loc.T("attachment_get_error_file_exists", map[string]any{"Path": path}),
		Hint:     loc.T("attachment_get_error_file_exists_hint"),
		ExitCode: output.ExitUsage,
	}
}

// convertSaveErrorは、中身の保存の失敗をCLIのエラーにする。
// 署名付きURLからの読み取りの失敗は通信の失敗 (同じコマンドの再実行で直りうる) とし、
// それ以外は保存先への書き込みの失敗としてwriteErrorで作る。
func convertSaveError(loc *i18n.Localizer, err error, writeError func(error) error) error {
	var networkErr *apiclient.DownloadNetworkError
	if errors.As(err, &networkErr) {
		return apiutil.ConvertError(loc, err)
	}

	return writeError(err)
}

// newWriteErrorは、保存先pathに書き込めなかったことを表すエラーを返す。
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

	message := loc.T("attachment_get_error_write", map[string]any{"Path": path, "Detail": detail.Error()})

	return newWriteFailedError(message, loc.T("attachment_get_error_write_hint"), err)
}

// newWriteFailedErrorは、翻訳済みのmessageとhintで、書き込みの失敗のエラーを返す。
func newWriteFailedError(message, hint string, err error) *output.Error {
	return &output.Error{
		Code:     output.CodeWriteFailed,
		Message:  message,
		Hint:     hint,
		ExitCode: output.ExitError,
		Err:      err,
	}
}

// saveFileは、rの中身をpathに保存し、書いたバイト数を返す。
// clobberでなければ既存のファイルを上書きせず、`fs.ErrExist` を返す。
// clobberなら同じディレクトリの一時ファイルに書いてから置き換え、取得に失敗しても既存のファイルを壊さない。
// 置き換えるファイルが通常のファイルなら、その権限を保つ。
// どちらも、失敗したときは書きかけのファイルを残さない。
func saveFile(path string, r io.Reader, clobber bool) (int64, error) {
	if !clobber {
		// O_EXCLは既存のファイルとシンボリックリンクがあれば失敗するため、確認の後に作られたファイルも上書きしない。
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, filePerm) // #nosec G302 G304 -- 保存先は利用者が指定したパスか、整えたファイル名。権限はfilePermを参照
		if err != nil {
			return 0, err
		}
		size, err := copyAndClose(file, r)
		if err != nil {
			_ = os.Remove(path)
			return 0, err
		}
		return size, nil
	}

	file, err := createTemp(filepath.Dir(path))
	if err != nil {
		return 0, err
	}
	size, err := copyAndClose(file, r)
	if err == nil {
		if info, statErr := os.Lstat(path); statErr == nil && info.Mode().IsRegular() {
			err = os.Chmod(file.Name(), info.Mode().Perm())
		}
	}
	if err == nil {
		err = os.Rename(file.Name(), path)
	}
	if err != nil {
		_ = os.Remove(file.Name())
		return 0, err
	}

	return size, nil
}

// createTempは、dirに一時ファイルを作る。
// os.CreateTempは権限を0600に固定するため、umaskに従うfilePermで作るよう名前を自分で選ぶ。
func createTemp(dir string) (*os.File, error) {
	var err error
	for range 10 {
		var file *os.File
		name := filepath.Join(dir, ".wikino-attachment-"+rand.Text())
		file, err = os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, filePerm) // #nosec G302 G304 -- 保存先のディレクトリに作る乱数の名前。権限はfilePermを参照
		if !errors.Is(err, fs.ErrExist) {
			return file, err
		}
	}

	return nil, err
}

func copyAndClose(file *os.File, r io.Reader) (int64, error) {
	size, err := io.Copy(file, r)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}

	return size, err
}

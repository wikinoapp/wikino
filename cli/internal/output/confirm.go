package output

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/iostreams"
)

// Confirmationは、取り消せない操作や公開の前に利用者へ示す確認の内容。
type Confirmation struct {
	// Changesは何が起きるかを1件ずつ表す文言。
	Changes []string
	// Commandは、承認を得た後に実行するコマンドの引数 (先頭はwikino)。
	// 確認した内容と実行する内容がずれないよう、確認のために取得した版を `--if-match` で固定して入れる。
	// トークンなどの秘密の値を含めてはならない。
	Command []string
}

// confirmationRequiredは、非対話のときに標準出力に出す確認プロトコルのJSONの形。
type confirmationRequired struct {
	Status         string   `json:"status"`
	Changes        []string `json:"changes"`
	ConfirmCommand string   `json:"confirm_command"`
}

// Confirmは、cの内容で操作を続けてよいかを確かめる。続けてよければnilを返す。
//
// 対話できるときは標準エラーに変更の内容を示して利用者に尋ね、承認されなければ終了コード9の
// エラーを返す。対話できないときはプロンプトで止まらず、何が起きるか (changes) と承認後に
// 実行するコマンド (confirm_command) を標準出力にJSONで出し、終了コード9で終える (確認プロトコル)。
// エージェントはchangesを利用者に見せ、承認を得てからconfirm_commandを実行する。
//
// `--yes` で確認を省く判断は呼び出し側が行い、そのときはConfirmを呼ばない。
func Confirm(ios *iostreams.IOStreams, loc *i18n.Localizer, c Confirmation) error {
	if !ios.CanPrompt() {
		var buf bytes.Buffer
		if err := encodeJSON(&buf, confirmationRequired{
			Status:         "confirmation_required",
			Changes:        c.Changes,
			ConfirmCommand: JoinCommand(c.Command),
		}, ios.IsStdoutTTY()); err != nil {
			return err
		}
		if _, err := buf.WriteTo(ios.Out); err != nil {
			return err
		}

		return &SilentError{ExitCode: ExitConfirmationRequired}
	}

	var prompt strings.Builder
	for _, change := range c.Changes {
		prompt.WriteString("- " + sanitizeLine(change) + "\n")
	}
	prompt.WriteString(loc.T("confirm_prompt") + " ")
	if _, err := fmt.Fprint(ios.ErrOut, prompt.String()); err != nil {
		return err
	}

	// 入力が閉じられた (Ctrl-Dなど) ときや読み取りに失敗したときは、読めた分だけで判断する。
	// 何も読めなければ承認されなかったものとして扱う。
	answer, _ := bufio.NewReader(ios.In).ReadString('\n')

	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return nil
	default:
		return &Error{
			Code:     CodeConfirmationDeclined,
			Message:  loc.T("error_confirmation_declined"),
			ExitCode: ExitConfirmationRequired,
		}
	}
}

// shellSafeArgは、シェルで引用せずに渡せる引数の文字。
var shellSafeArg = regexp.MustCompile(`^[A-Za-z0-9_@%+=:,./-]+$`)

// JoinCommandは、argsをPOSIXのシェルにそのまま貼り付けて実行できる1行にする。
// 引用が要る引数は単一引用符で囲む。ETag (`"d41d8cd9"`) のように引用符を含む値を、
// エージェントが書き写すときに崩さないためである。
func JoinCommand(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		if shellSafeArg.MatchString(arg) {
			quoted[i] = arg
			continue
		}
		quoted[i] = "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
	}

	return strings.Join(quoted, " ")
}

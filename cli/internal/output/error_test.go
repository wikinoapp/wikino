package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/i18n"
)

func TestWriteError(t *testing.T) {
	t.Parallel()

	preconditionFailed := &Error{
		Code:     "precondition_failed",
		Message:  "ページはその後に更新されています。",
		Hint:     "`wikino page get 12` で最新の内容とETagを取得してください。",
		Status:   412,
		ExitCode: ExitConflict,
		Err:      errors.New("内部の詳細"),
	}

	tests := []struct {
		name   string
		err    error
		asJSON bool
		want   string
	}{
		{
			name: "人間向けはメッセージとヒントを1行ずつ出す",
			err:  preconditionFailed,
			want: "ページはその後に更新されています。\n`wikino page get 12` で最新の内容とETagを取得してください。\n",
		},
		{
			name:   "JSONはcode・message・hint・statusを1行で出す",
			err:    preconditionFailed,
			asJSON: true,
			want:   `{"error":{"code":"precondition_failed","message":"ページはその後に更新されています。","hint":"` + "`wikino page get 12` で最新の内容とETagを取得してください。" + `","status":412}}` + "\n",
		},
		{
			name:   "包まれたエラーも取り出して出す",
			err:    fmt.Errorf("ページの更新: %w", preconditionFailed),
			asJSON: true,
			want:   `{"error":{"code":"precondition_failed","message":"ページはその後に更新されています。","hint":"` + "`wikino page get 12` で最新の内容とETagを取得してください。" + `","status":412}}` + "\n",
		},
		{
			name:   "APIに由来しないエラーはstatusを省く",
			err:    &Error{Code: CodeUsage, Message: "不明なフラグです: --x", Hint: "`wikino --help` で使い方を確認してください。", ExitCode: ExitUsage},
			asJSON: true,
			want:   `{"error":{"code":"usage_error","message":"不明なフラグです: --x","hint":"` + "`wikino --help` で使い方を確認してください。" + `"}}` + "\n",
		},
		{
			name: "人間向けはメッセージとヒントの制御文字を空白に置き換える",
			err:  &Error{Code: CodeJQFailed, Message: "error: a\x1b[2Jb\u202ec\rd", Hint: "ヒント\n2行目", ExitCode: ExitError},
			want: "error: a [2Jb c d\nヒント 2行目\n",
		},
		{
			name:   "JSONはメッセージとヒントの制御文字をエスケープして元の値を保つ",
			err:    &Error{Code: CodeJQFailed, Message: "error: a\x1b[2Jb", Hint: "ヒント\n2行目", ExitCode: ExitError},
			asJSON: true,
			want:   `{"error":{"code":"jq_failed","message":"error: a\u001b[2Jb","hint":"ヒント\n2行目"}}` + "\n",
		},
		{
			name: "ヒントの無いエラーはメッセージだけを出す",
			err:  errors.New("通信に失敗しました"),
			want: "通信に失敗しました\n",
		},
		{
			name:   "CLIのエラー型でないエラーはunexpected_errorとして出す",
			err:    errors.New("通信に失敗しました"),
			asJSON: true,
			want:   `{"error":{"code":"unexpected_error","message":"通信に失敗しました"}}` + "\n",
		},
		{
			name:   "HTMLの記号をエスケープしない",
			err:    errors.New("<a> & <b>"),
			asJSON: true,
			want:   `{"error":{"code":"unexpected_error","message":"<a> & <b>"}}` + "\n",
		},
		{
			name: "出力を済ませたエラーは何も出さない",
			err:  &SilentError{ExitCode: ExitConfirmationRequired},
			want: "",
		},
		{
			name: "エラーが無ければ何も出さない",
			err:  nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			if err := WriteError(&buf, tt.err, tt.asJSON); err != nil {
				t.Fatalf("書き込みのエラーを期待しなかったが %v だった", err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("出力 = %q、期待値 = %q", got, tt.want)
			}
			if strings.Contains(buf.String(), "内部の詳細") {
				t.Errorf("原因のエラーを出さないことを期待したが、含まれていた: %q", buf.String())
			}
		})
	}
}

func TestExitCodeOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil", err: nil, want: ExitOK},
		{name: "CLIのエラー", err: &Error{ExitCode: ExitNotFound}, want: ExitNotFound},
		{name: "包まれたCLIのエラー", err: fmt.Errorf("取得: %w", &Error{ExitCode: ExitAuth}), want: ExitAuth},
		{name: "出力を済ませたエラー", err: &SilentError{ExitCode: ExitConfirmationRequired}, want: ExitConfirmationRequired},
		{name: "分類しないエラー", err: errors.New("失敗"), want: ExitError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ExitCodeOf(tt.err); got != tt.want {
				t.Errorf("ExitCodeOf() = %d、期待値 = %d", got, tt.want)
			}
		})
	}
}

func TestExitCodeTable(t *testing.T) {
	t.Parallel()

	for _, lang := range []string{i18n.LangJa, i18n.LangEn} {
		t.Run(lang, func(t *testing.T) {
			t.Parallel()

			table := ExitCodeTable(i18n.NewLocalizer(lang))
			lines := strings.Split(table, "\n")
			if len(lines) != len(exitCodes) {
				t.Fatalf("行数 = %d、期待値 = %d:\n%s", len(lines), len(exitCodes), table)
			}
			for i, e := range exitCodes {
				prefix := "  " + strconv.Itoa(e.code) + " "
				if !strings.HasPrefix(lines[i], prefix) {
					t.Errorf("%d行目 = %q、%q で始まることを期待した", i+1, lines[i], prefix)
				}
				if strings.Contains(lines[i], e.messageID) {
					t.Errorf("%d行目の文言が翻訳されていない: %q", i+1, lines[i])
				}
			}
		})
	}
}

// 確認プロトコルとエラーのJSONは、エージェントが読む公開の契約のため、キーの名前を固定する。
func TestErrorJSONKeys(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	if err := WriteError(&buf, &Error{Code: "c", Message: "m", Hint: "h", Status: 404}, true); err != nil {
		t.Fatal(err)
	}

	var got map[string]map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSONとして読めません: %v", err)
	}
	for _, key := range []string{"code", "message", "hint", "status"} {
		if _, ok := got["error"][key]; !ok {
			t.Errorf("errorに %s が無い: %s", key, buf.String())
		}
	}
}

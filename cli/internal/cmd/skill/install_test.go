package skill

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/cmdtest"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
	skillfiles "github.com/wikinoapp/wikino/cli/internal/skill"
)

// newEnvは、ホームディレクトリを一時ディレクトリにした実行環境と、そのホームディレクトリを返す。
// ホームディレクトリを決めずに実行すると、テストを実行した人のスキルのディレクトリに書き出してしまう。
func newEnv(t *testing.T) (*cmdtest.Env, string) {
	t.Helper()

	e := cmdtest.New(t)
	home := t.TempDir()
	e.Vars["HOME"] = home

	return e, home
}

// assertInstalledは、dirにlangの言語のスキルのすべてのファイルが同じ中身で書き出されたことを確かめる。
func assertInstalled(t *testing.T, dir, lang string) []string {
	t.Helper()

	files, err := skillfiles.Files(lang)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(files))
	for _, file := range files {
		names = append(names, file.Name)
		got, err := os.ReadFile(filepath.Join(dir, file.Name)) // #nosec G304 -- テストが作った一時ディレクトリのファイル
		if err != nil {
			t.Errorf("%sを読めない: %v", file.Name, err)
			continue
		}
		if string(got) != string(file.Content) {
			t.Errorf("%sの中身が埋め込んだスキルと違う", file.Name)
		}
	}

	return names
}

func TestInstall_DefaultDir(t *testing.T) {
	t.Parallel()

	e, home := newEnv(t)

	if err := e.Run(NewCmdSkill, "skill", "install"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	dir := filepath.Join(home, ".claude", "skills", "wikino")
	names := assertInstalled(t, dir, i18n.LangJa)

	var got installItem
	cmdtest.Decode(t, e.Stdout.String(), &got)
	want := installItem{Path: dir, Lang: i18n.LangJa, Files: names}
	if got.Path != want.Path || got.Lang != want.Lang || !slices.Equal(got.Files, want.Files) {
		t.Errorf("出力 = %+v、期待値 = %+v", got, want)
	}
	if e.Stderr.Len() > 0 {
		t.Errorf("標準エラーに何も出さないことを期待したが %q だった", e.Stderr.String())
	}

	info, err := os.Stat(filepath.Join(dir, skillfiles.EntryFileName))
	if err != nil {
		t.Fatal(err)
	}
	// 秘密の値を含まないため、0600にしない (os.CreateTempの既定の権限を残さない)。
	if perm := info.Mode().Perm(); perm&0o044 == 0 {
		t.Errorf("SKILL.mdの権限 = %o、グループと他の人も読めることを期待した", perm)
	}
}

func TestInstall_ClaudeConfigDir(t *testing.T) {
	t.Parallel()

	e, _ := newEnv(t)
	configDir := t.TempDir()
	e.Vars["CLAUDE_CONFIG_DIR"] = configDir

	if err := e.Run(NewCmdSkill, "skill", "install", "--jq", ".path"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	dir := filepath.Join(configDir, "skills", "wikino")
	assertInstalled(t, dir, i18n.LangJa)
	if got := strings.TrimSpace(e.Stdout.String()); got != dir {
		t.Errorf("出力 = %s、期待値 = %s", got, dir)
	}
}

func TestInstall_Dir(t *testing.T) {
	t.Parallel()

	e, home := newEnv(t)
	skillsDir := t.TempDir()
	dir := filepath.Join(skillsDir, "wikino")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// 前に書き出したスキルのファイルは上書きし、利用者が置いたほかのファイルは残す。
	if err := os.WriteFile(filepath.Join(dir, skillfiles.EntryFileName), []byte("古いスキル"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "memo.md"), []byte("メモ"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := e.Run(NewCmdSkill, "skill", "install", "--dir", skillsDir); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	assertInstalled(t, dir, i18n.LangJa)
	if got, err := os.ReadFile(filepath.Join(dir, "memo.md")); err != nil || string(got) != "メモ" { // #nosec G304 -- テストが作った一時ディレクトリのファイル
		t.Errorf("ほかのファイルが残ることを期待したが、中身 = %q、エラー = %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude")); err == nil {
		t.Error("`--dir` を指定したのに既定の書き出し先にも書き出した")
	}
	// 書き込みの途中の一時ファイルを残さない。
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			t.Errorf("一時ファイル %s が残っている", entry.Name())
		}
	}
}

func TestInstall_English(t *testing.T) {
	t.Parallel()

	e, home := newEnv(t)
	e.Vars["WIKINO_LANG"] = i18n.LangEn

	if err := e.Run(NewCmdSkill, "skill", "install", "--fields", "lang"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	assertInstalled(t, filepath.Join(home, ".claude", "skills", "wikino"), i18n.LangEn)
	if got := strings.TrimSpace(e.Stdout.String()); got != `{"lang":"en"}` {
		t.Errorf("出力 = %s", got)
	}
}

func TestInstall_Human(t *testing.T) {
	t.Parallel()

	e, home := newEnv(t)

	if err := e.RunTTY(NewCmdSkill, "skill", "install"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	dir := filepath.Join(home, ".claude", "skills", "wikino")
	want := "書き出し先: " + dir + "\n" +
		"言語:       ja\n" +
		"ファイル:   SKILL.md, draft-and-publish.md, edit-page.md, login.md, read-page.md\n"
	if got := e.Stdout.String(); got != want {
		t.Errorf("出力 = %q、期待値 = %q", got, want)
	}
}

func TestInstall_WriteFailed(t *testing.T) {
	t.Parallel()

	e, _ := newEnv(t)
	// スキルを置くディレクトリにファイルを指定すると、その下にディレクトリを作れない。
	notDir := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(notDir, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	err := e.Run(NewCmdSkill, "skill", "install", "--dir", notDir)
	cliErr := cmdtest.AssertCLIError(t, err, output.CodeWriteFailed, output.ExitError)
	if !strings.Contains(cliErr.Message, filepath.Join(notDir, "wikino")) {
		t.Errorf("メッセージに書き出し先を含むことを期待したが %q だった", cliErr.Message)
	}
	if cliErr.Hint == "" {
		t.Error("ヒントを期待した")
	}
}

func TestInstall_UsageErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "引数がある", args: []string{"skill", "install", "extra"}},
		{name: "未知の項目", args: []string{"skill", "install", "--fields", "nosuchfield"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e, home := newEnv(t)
			err := e.Run(NewCmdSkill, tt.args...)
			_ = cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
			if _, err := os.Stat(filepath.Join(home, ".claude")); err == nil {
				t.Error("使い方の誤りなのに書き出した")
			}
		})
	}
}

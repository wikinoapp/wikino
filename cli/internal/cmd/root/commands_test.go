package root

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/wikinoapp/wikino/cli/internal/cmdtest"
	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/iostreams"
	"github.com/wikinoapp/wikino/cli/internal/skill"
)

var langs = []string{i18n.LangJa, i18n.LangEn}

// newTestRootは、実在するすべてのコマンドを持つルートのコマンドをlangの言語で組み立てる。
func newTestRoot(t *testing.T, lang string) *cobra.Command {
	t.Helper()

	ios, _, _, _ := iostreams.Test()

	return newCmdRoot(newTestFactory(i18n.NewLocalizer(lang), ios), func(error) {})
}

// allCommandsは、ルートを含めて、ヘルプに出すすべてのコマンドをヘルプのコマンドの一覧と同じ順に返す。
func allCommands(cmd *cobra.Command) []*cobra.Command {
	cmds := []*cobra.Command{cmd}
	for _, sub := range cmd.Commands() {
		if sub.IsAvailableCommand() {
			cmds = append(cmds, allCommands(sub)...)
		}
	}

	return cmds
}

// untranslatedは、翻訳が見つからずに翻訳キーのまま残った文言に一致する。
var untranslated = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)+$`)

func TestAllCommandsHaveHelpSections(t *testing.T) {
	t.Parallel()

	for _, lang := range langs {
		for _, cmd := range allCommands(newTestRoot(t, lang)) {
			// サブテストは並行に実行しない。LocalFlagsは親のコマンドのフラグの集まりを書き換えるため、
			// 同じルートの下のコマンドを並行に調べるとデータ競合になる。
			t.Run(lang+"/"+cmd.CommandPath(), func(t *testing.T) {
				sections := map[string]string{
					"Short": cmd.Short,
					"Long":  cmd.Long,
					"例":     cmd.Example,
					"出力":    cmd.Annotations[cmdutil.AnnotationOutput],
					"エラー":   cmd.Annotations[cmdutil.AnnotationErrors],
				}
				for name, text := range sections {
					if strings.TrimSpace(text) == "" {
						t.Errorf("%sの節が無い", name)
					}
					if untranslated.MatchString(text) {
						t.Errorf("%sの節が翻訳されていない: %q", name, text)
					}
				}

				cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
					if strings.TrimSpace(f.Usage) == "" || untranslated.MatchString(f.Usage) {
						t.Errorf("--%sの説明が無いか翻訳されていない: %q", f.Name, f.Usage)
					}
				})
			})
		}
	}
}

func TestHelpGolden(t *testing.T) {
	t.Parallel()

	for _, lang := range langs {
		t.Run(lang, func(t *testing.T) {
			t.Parallel()

			var text, jsonHelp strings.Builder
			for _, cmd := range allCommands(newTestRoot(t, lang)) {
				path := strings.Fields(cmd.CommandPath())[1:]

				text.WriteString("=== " + cmd.CommandPath() + " --help\n")
				text.WriteString(executeHelp(t, lang, append(path, "--help")))
				text.WriteString("\n")

				var indented bytes.Buffer
				if err := json.Indent(&indented, []byte(executeHelp(t, lang, append(path, "--help", "--json"))), "", "  "); err != nil {
					t.Fatalf("%sのJSONのヘルプを読めない: %v", cmd.CommandPath(), err)
				}
				jsonHelp.WriteString("=== " + cmd.CommandPath() + " --help --json\n")
				jsonHelp.WriteString(indented.String())
				jsonHelp.WriteString("\n")
			}

			// JSONのヘルプのゴールデンは、コマンドとフラグの削除・改名を検出するスナップショットを兼ねる。
			cmdtest.AssertGolden(t, filepath.Join("help", lang), text.String())
			cmdtest.AssertGolden(t, filepath.Join("help", lang+".json"), jsonHelp.String())
		})
	}
}

// executeHelpは、argsで実在するコマンドのヘルプを出し、標準出力を返す。
func executeHelp(t *testing.T, lang string, args []string) string {
	t.Helper()

	ios, _, stdout, stderr := iostreams.Test()
	if _, err := Execute(args, newTestFactory(i18n.NewLocalizer(lang), ios)); err != nil {
		t.Fatalf("%vのヘルプでエラーを期待しなかったが %v だった", args, err)
	}
	if stderr.Len() > 0 {
		t.Errorf("%vのヘルプで標準エラーに何も出さないことを期待したが %q だった", args, stderr.String())
	}

	return stdout.String()
}

func TestCommandTableIsUpToDate(t *testing.T) {
	t.Parallel()

	for _, lang := range langs {
		content := skillFile(t, lang, skill.EntryFileName)
		want, err := skill.ReplaceCommandTable(content, CommandTable(lang))
		if err != nil {
			t.Fatalf("%sのSKILL.mdに一覧表の印が無い: %v", lang, err)
		}
		if content != want {
			t.Errorf("%sのSKILL.mdのコマンドの一覧表がコマンド定義と違う。`make -C cli skill-generate` で生成し直してください\n--- 生成したSKILL.md\n%s\n--- ファイル\n%s", lang, want, content)
		}
	}
}

func TestCommandReferencesExist(t *testing.T) {
	t.Parallel()

	for _, lang := range langs {
		root := newTestRoot(t, lang)
		messages := readMessages(t, lang)

		count := 0
		for _, key := range slices.Sorted(mapsKeys(messages)) {
			for _, ref := range extractCommandReferences(messages[key]) {
				count++
				if err := resolveCommandReference(root, ref); err != "" {
					t.Errorf("%s.tomlの%sに書いた `%s` が実在しない: %s", lang, key, ref, err)
				}
			}
		}

		// 抜き出しの誤りで検査が空回りしていないことを確かめる。
		if count < 100 {
			t.Errorf("%s.tomlから抜き出したコマンドが %d 件しかない", lang, count)
		}
	}
}

func TestSkillCommandReferencesExist(t *testing.T) {
	t.Parallel()

	for _, lang := range langs {
		root := newTestRoot(t, lang)
		files, err := skill.Files(lang)
		if err != nil {
			t.Fatal(err)
		}

		count := 0
		for _, file := range files {
			for _, ref := range extractCommandReferences(string(file.Content)) {
				count++
				if err := resolveCommandReference(root, ref); err != "" {
					t.Errorf("%s/%sに書いた `%s` が実在しない: %s", lang, file.Name, ref, err)
				}
			}
		}

		// 抜き出しの誤りで検査が空回りしていないことを確かめる。
		if count < 50 {
			t.Errorf("%sのスキルから抜き出したコマンドが %d 件しかない", lang, count)
		}
	}
}

// TestSkillConceptSectionHasNoCommandsは、SKILL.mdの最初の節 (Wikinoの概念と扱うときの原則) に
// CLIのコマンドとフラグが無いことを確かめる。この節はリモートMCPサーバーのinstructionsにも使うため。
func TestSkillConceptSectionHasNoCommands(t *testing.T) {
	t.Parallel()

	for _, lang := range langs {
		section := firstSection(skillFile(t, lang, skill.EntryFileName))
		if strings.TrimSpace(section) == "" {
			t.Fatalf("%sのSKILL.mdに節が無い", lang)
		}
		if refs := extractCommandReferences(section); len(refs) > 0 {
			t.Errorf("%sのSKILL.mdの最初の節にコマンドがある: %q", lang, refs)
		}
		if flag := regexp.MustCompile("`--[a-z]").FindString(section); flag != "" {
			t.Errorf("%sのSKILL.mdの最初の節にフラグがある", lang)
		}
	}
}

// firstSectionは、Markdownのcontentの最初の `## ` の見出しから、次の `## ` の見出しの前までを返す。
func firstSection(content string) string {
	_, rest, ok := strings.Cut(content, "\n## ")
	if !ok {
		return ""
	}
	section, _, _ := strings.Cut(rest, "\n## ")

	return section
}

// skillFileは、langの言語のスキルのファイルnameの中身を返す。
func skillFile(t *testing.T, lang, name string) string {
	t.Helper()

	files, err := skill.Files(lang)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if file.Name == name {
			return string(file.Content)
		}
	}
	t.Fatalf("%sのスキルに%sが無い", lang, name)

	return ""
}

func TestExtractCommandReferences(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "バッククォートで囲んだコマンド",
			text: "`wikino page list` で番号を確かめ、`wikino page get <番号>` で読んでください。`wikino_etag` はキーです。",
			want: []string{"wikino page list", "wikino page get <番号>"},
		},
		{
			name: "例の行とパイプ",
			text: "  # 本文を標準入力で渡す\n  cat minutes.md | wikino page create --title '議事録' --body-file -\n  wikino page get 12 --md > page-12.md",
			want: []string{"wikino page create --title '議事録' --body-file -", "wikino page get 12 --md > page-12.md"},
		},
		{
			name: "コマンドでない語",
			text: "設定は `$XDG_CONFIG_HOME/wikino/` に保存します。`{{.Command}} --help` で確認できます。",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := extractCommandReferences(tt.text); !slices.Equal(got, tt.want) {
				t.Errorf("抜き出したコマンド = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}

func TestResolveCommandReference(t *testing.T) {
	t.Parallel()

	root := newTestRoot(t, i18n.LangJa)

	tests := []struct {
		ref     string
		wantErr bool
	}{
		{ref: "wikino"},
		{ref: "wikino page"},
		{ref: "wikino page get 12 --md > page-12.md"},
		{ref: "wikino --space team page list --topic 3 --title '議事録'"},
		{ref: "wikino page list --topic=3"},
		{ref: "wikino attachment get abc -O -"},
		{ref: "wikino draft revision get 12 <revision_id>"},
		{ref: "wikino auth login {{.Space}}"},
		{ref: "wikino <command> --help"},
		{ref: "wikino page <command> --help"},
		{ref: "wikino pages list", wantErr: true},
		{ref: "wikino page view 12", wantErr: true},
		{ref: "wikino page get 12 --markdown", wantErr: true},
		{ref: "wikino page get 12 -x", wantErr: true},
		{ref: "wikino page list --dry-run", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			t.Parallel()

			if err := resolveCommandReference(root, tt.ref); (err != "") != tt.wantErr {
				t.Errorf("resolveCommandReference(%q) = %q、誤りを期待したか = %v", tt.ref, err, tt.wantErr)
			}
		})
	}
}

// readMessagesは、langの翻訳ファイルの翻訳キーと文言を返す。
func readMessages(t *testing.T, lang string) map[string]string {
	t.Helper()

	var file map[string]struct {
		Other string `toml:"other"`
	}
	path := filepath.Join("..", "..", "i18n", "locales", lang+".toml")
	if _, err := toml.DecodeFile(path, &file); err != nil {
		t.Fatalf("%sを読めない: %v", path, err)
	}

	messages := make(map[string]string, len(file))
	for key, message := range file {
		messages[key] = message.Other
	}

	return messages
}

func mapsKeys(m map[string]string) func(yield func(string) bool) {
	return func(yield func(string) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}

// shellSeparatorは、1行に複数のコマンドを並べるシェルの区切り。
var shellSeparator = regexp.MustCompile(`\|\||\||&&|;|\$\(`)

// extractCommandReferencesは、文言に書いた `wikino ...` のコマンドを抜き出す。
// バッククォートで囲んだ部分と、例のようにそのまま書いた行の両方から、シェルの区切りで分けた
// コマンドのうち `wikino` で始まるものを取り出す。
func extractCommandReferences(text string) []string {
	var refs []string
	for line := range strings.Lines(strings.ReplaceAll(text, "`", "\n")) {
		for _, part := range shellSeparator.Split(line, -1) {
			part = strings.TrimSpace(part)
			if part == "wikino" || strings.HasPrefix(part, "wikino ") {
				refs = append(refs, part)
			}
		}
	}

	return refs
}

// resolveCommandReferenceは、refのコマンドとフラグが実在するかを確かめ、実在しなければ理由を返す。
// 位置引数の数と値は確かめない。文言では `<番号>` や `{{.Space}}` のような置き換える値を書くため。
func resolveCommandReference(root *cobra.Command, ref string) string {
	cmd := root
	positional := false

	tokens := shellWords(ref)[1:]
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]

		// リダイレクトとコメントより後ろはコマンドの引数ではない。
		if token == ">" || token == ">>" || token == "<" || token == "2>&1" || strings.HasPrefix(token, "#") {
			break
		}

		if strings.HasPrefix(token, "-") && token != "-" {
			flag, ok := lookupFlag(cmd, token)
			if !ok {
				return "`" + cmd.CommandPath() + "` にフラグ " + token + " が無い"
			}
			// 値を取るフラグで、値を `=` で繋いでいなければ、次の語を値として読み飛ばす。
			if flag.NoOptDefVal == "" && !strings.Contains(token, "=") && !isShortWithValue(token) {
				i++
			}
			continue
		}

		if !positional {
			if sub := findSubcommand(cmd, token); sub != nil {
				cmd = sub
				continue
			}
			// `wikino <command> --help` のように、サブコマンドを置き換える値で書いたものは、どのサブコマンドとも確かめない。
			if cmdutil.IsGroup(cmd) && strings.HasPrefix(token, "<") {
				break
			}
			if cmdutil.IsGroup(cmd) {
				return "`" + cmd.CommandPath() + "` にサブコマンド " + token + " が無い"
			}
		}
		positional = true
	}

	return ""
}

func findSubcommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, sub := range cmd.Commands() {
		if sub.IsAvailableCommand() && sub.Name() == name {
			return sub
		}
	}

	return nil
}

// lookupFlagは、cmdのフラグ (継承したものを含む) からtoken (`--name`・`--name=value`・`-n`) を探す。
func lookupFlag(cmd *cobra.Command, token string) (*pflag.Flag, bool) {
	var flag *pflag.Flag
	if name, ok := strings.CutPrefix(token, "--"); ok {
		name, _, _ = strings.Cut(name, "=")
		for _, fs := range flagSets(cmd) {
			if flag = fs.Lookup(name); flag != nil {
				break
			}
		}
	} else {
		for _, fs := range flagSets(cmd) {
			if flag = fs.ShorthandLookup(token[1:2]); flag != nil {
				break
			}
		}
	}

	return flag, flag != nil
}

// flagSetsは、cmdで使えるフラグの集まりを返す。コマンドを実行する前は、cmd自身の永続フラグと
// 親から継承するフラグがFlagsに入っていないため、別に引く。
func flagSets(cmd *cobra.Command) []*pflag.FlagSet {
	return []*pflag.FlagSet{cmd.Flags(), cmd.PersistentFlags(), cmd.InheritedFlags()}
}

// isShortWithValueは、短縮フラグに値を続けて書いたもの (`-Ofile`) かを返す。
func isShortWithValue(token string) bool {
	return !strings.HasPrefix(token, "--") && len(token) > 2
}

// shellWordsは、sをシェルと同じく空白で語に分ける。引用符で囲んだ部分は1つの語として扱う。
func shellWords(s string) []string {
	var words []string
	var word strings.Builder
	inWord := false
	var quote rune

	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				word.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
			inWord = true
		case r == ' ' || r == '\t' || r == '\n':
			if inWord {
				words = append(words, word.String())
				word.Reset()
				inWord = false
			}
		default:
			word.WriteRune(r)
			inWord = true
		}
	}
	if inWord {
		words = append(words, word.String())
	}

	return words
}

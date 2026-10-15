package root

import (
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/text/width"

	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/iostreams"
)

// CommandTableは、スキル (SKILL.md) に載せるコマンドの一覧表を、langの言語のMarkdownで返す。
// 一覧表はSKILL.mdの印の間に差し込む (skill.ReplaceCommandTable)。
//
// 一覧表はcobraのコマンド定義だけを情報源にして生成し、手で書いた表とヘルプがずれないようにする。
// 表の桁はoxfmtで整形したときと同じく揃え、`make fmt` で生成物が書き換わらないようにする。
func CommandTable(lang string) string {
	loc := i18n.NewLocalizer(lang)
	ios, _, _, _ := iostreams.Test()
	root := newCmdRoot(&cmdutil.Factory{
		IOStreams: ios,
		Localizer: loc,
		Getenv:    func(string) string { return "" },
	}, func(error) {})

	rows := [][]string{{loc.T("command_table_heading_command"), loc.T("command_table_heading_description")}}
	for _, cmd := range RunnableCommands(root) {
		rows = append(rows, []string{"`" + commandLine(cmd) + "`", escapeTableCell(cmd.Short)})
	}

	var b strings.Builder
	writeTable(&b, rows)

	return b.String()
}

// RunnableCommandsは、cmdの下にある実行できるコマンドを、ヘルプのコマンドの一覧と同じ順に返す。
// サブコマンドを束ねるだけのコマンドと、ヘルプに出さないコマンドは含めない。
func RunnableCommands(cmd *cobra.Command) []*cobra.Command {
	var cmds []*cobra.Command
	for _, sub := range cmd.Commands() {
		if !sub.IsAvailableCommand() {
			continue
		}
		if !cmdutil.IsGroup(sub) {
			cmds = append(cmds, sub)
		}
		cmds = append(cmds, RunnableCommands(sub)...)
	}

	return cmds
}

// commandLineは、コマンドの完全な名前に位置引数を続けたもの (例: wikino page get <page_number>) を返す。
func commandLine(cmd *cobra.Command) string {
	_, args, _ := strings.Cut(cmd.Use, " ")
	if args == "" {
		return cmd.CommandPath()
	}

	return cmd.CommandPath() + " " + args
}

func escapeTableCell(s string) string {
	return strings.ReplaceAll(s, "|", `\|`)
}

// writeTableは、rowsの先頭の行を見出しにしたMarkdownの表を書く。
func writeTable(b *strings.Builder, rows [][]string) {
	widths := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, cell := range row {
			widths[i] = max(widths[i], displayWidth(cell))
		}
	}

	writeRow := func(cells []string) {
		b.WriteString("|")
		for i, cell := range cells {
			b.WriteString(" " + cell + strings.Repeat(" ", widths[i]-displayWidth(cell)) + " |")
		}
		b.WriteString("\n")
	}

	writeRow(rows[0])
	separators := make([]string, len(widths))
	for i, w := range widths {
		separators[i] = strings.Repeat("-", w)
	}
	writeRow(separators)
	for _, row := range rows[1:] {
		writeRow(row)
	}
}

// displayWidthは、等幅の表示で文字列が占める桁数を返す。全角の文字は2桁と数える。
func displayWidth(s string) int {
	n := 0
	for _, r := range s {
		switch width.LookupRune(r).Kind() {
		case width.EastAsianWide, width.EastAsianFullwidth:
			n += 2
		default:
			n++
		}
	}

	return n
}

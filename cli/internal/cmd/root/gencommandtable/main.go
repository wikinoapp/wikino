// gencommandtableは、スキル (SKILL.md) に載せるコマンドの一覧表を、cobraのコマンド定義から生成する開発用のコマンド。
//
// 対応する言語ごとに、-dirのディレクトリの `{言語}/SKILL.md` の印の間を一覧表で書き換える。
// 生成物とコマンド定義のずれはrootパッケージのテストで検出する。
//
// 使い方: go run ./internal/cmd/root/gencommandtable -dir internal/skill
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wikinoapp/wikino/cli/internal/cmd/root"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/skill"
)

func main() {
	dir := flag.String("dir", "", "スキルのディレクトリ")
	flag.Parse()

	if *dir == "" {
		fmt.Fprintln(os.Stderr, "-dirを指定してください")
		os.Exit(2)
	}

	for _, lang := range []string{i18n.LangJa, i18n.LangEn} {
		path := filepath.Join(*dir, lang, skill.EntryFileName)
		content, err := os.ReadFile(path) // #nosec G304 -- 開発者が指定したスキルのディレクトリのファイル
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sを読めません: %v\n", path, err)
			os.Exit(1)
		}
		replaced, err := skill.ReplaceCommandTable(string(content), root.CommandTable(lang))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sに一覧表を差し込めません: %v\n", path, err)
			os.Exit(1)
		}
		if err := os.WriteFile(path, []byte(replaced), 0o644); err != nil { // #nosec G306 G703 -- リポジトリにコミットする生成物
			fmt.Fprintf(os.Stderr, "%sを書き出せません: %v\n", path, err)
			os.Exit(1)
		}
	}
}

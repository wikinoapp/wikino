// Package skillは、コーディングエージェント向けのスキル (SKILL.mdと場面ごとの手順書) をバイナリに埋め込んで提供する。
//
// スキルは日本語 (ja/) を正本とし、英語 (en/) をその翻訳として置く。
// 各言語のディレクトリは同じ名前のファイルを持ち、`wikino skill install` は表示言語のディレクトリを書き出す。
// SKILL.mdのコマンドの一覧表は、`make -C cli skill-generate` でコマンド定義から生成する。
package skill

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
)

// Nameはスキルの名前。SKILL.mdのfrontmatterの `name` と、書き出すディレクトリの名前に使う。
const Name = "wikino"

// EntryFileNameは、エージェントが最初に読み込むファイルの名前。
const EntryFileName = "SKILL.md"

// コマンドの一覧表を生成して差し込む、SKILL.mdの印。
const (
	CommandTableBegin = "<!-- BEGIN commands -->"
	CommandTableEnd   = "<!-- END commands -->"
)

//go:embed ja en
var files embed.FS

// Filesは、langの言語のスキルのファイル名と中身を、ファイル名の順に返す。
func Files(lang string) ([]File, error) {
	entries, err := fs.ReadDir(files, lang)
	if err != nil {
		return nil, fmt.Errorf("スキルの言語 %s が無い: %w", lang, err)
	}

	result := make([]File, 0, len(entries))
	for _, entry := range entries {
		content, err := fs.ReadFile(files, lang+"/"+entry.Name())
		if err != nil {
			return nil, err
		}
		result = append(result, File{Name: entry.Name(), Content: content})
	}

	return result, nil
}

// Fileは、スキルの1つのファイル。
type File struct {
	Name    string
	Content []byte
}

// ReplaceCommandTableは、SKILL.mdの中身contentの印の間をtableに置き換えたものを返す。
// 表の前後には、oxfmtで整形したときと同じく空行を置く。
func ReplaceCommandTable(content, table string) (string, error) {
	before, rest, ok := strings.Cut(content, CommandTableBegin+"\n")
	if !ok {
		return "", fmt.Errorf("%s が無い", CommandTableBegin)
	}
	_, after, ok := strings.Cut(rest, CommandTableEnd)
	if !ok {
		return "", fmt.Errorf("%s が無い", CommandTableEnd)
	}

	return before + CommandTableBegin + "\n\n" + table + "\n" + CommandTableEnd + after, nil
}

package mcpserver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// skillPathは、instructionsの元にするCLIのスキル (日本語の正本) のパス。このパッケージのディレクトリから辿る
var skillPath = filepath.Join("..", "..", "..", "cli", "internal", "skill", "ja", "SKILL.md")

// instructionsSectionは、スキルの本文skillから「Wikinoについて」の節を、`make mcp-instructions-generate` と
// 同じ規則 (HTMLのコメントとその後ろの空行、末尾の空行を除く) で取り出す
func instructionsSection(skill string) string {
	var lines []string
	inSection, afterComment := false, false
	for _, line := range strings.Split(skill, "\n") {
		if strings.HasPrefix(line, "## ") {
			inSection = line == "## Wikinoについて"
		}
		if !inSection {
			continue
		}
		if strings.HasPrefix(line, "<!--") {
			afterComment = true
			continue
		}
		if afterComment && line == "" {
			afterComment = false
			continue
		}
		afterComment = false
		lines = append(lines, line)
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
}

func TestInstructions_CLIのスキルの節とずれていない(t *testing.T) {
	t.Parallel()

	skill, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("CLIのスキルの読み込みに失敗: %v", err)
	}
	want := instructionsSection(string(skill))
	if want == "\n" {
		t.Fatal("CLIのスキルに「Wikinoについて」の節が無い")
	}
	if instructions != want {
		t.Errorf("instructions.mdがCLIのスキルの「Wikinoについて」の節とずれている。`make mcp-instructions-generate` で写し直す\n得た値 = %q\n期待値 = %q", instructions, want)
	}
}

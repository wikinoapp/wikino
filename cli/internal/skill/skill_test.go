package skill

import (
	"regexp"
	"slices"
	"strings"
	"testing"
)

var langs = []string{"ja", "en"}

func fileNames(t *testing.T, lang string) []string {
	t.Helper()

	files, err := Files(lang)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(files))
	for _, file := range files {
		names = append(names, file.Name)
	}

	return names
}

func TestLangsHaveSameFiles(t *testing.T) {
	t.Parallel()

	ja := fileNames(t, "ja")
	if !slices.Contains(ja, EntryFileName) {
		t.Fatalf("jaのスキルに%sが無い: %q", EntryFileName, ja)
	}
	// 英語版は日本語版の翻訳のため、同じ名前のファイルを持つ。
	if en := fileNames(t, "en"); !slices.Equal(ja, en) {
		t.Errorf("jaとenのファイルが違う: ja = %q、en = %q", ja, en)
	}
}

func TestFilesUnknownLang(t *testing.T) {
	t.Parallel()

	if _, err := Files("fr"); err == nil {
		t.Error("未知の言語でエラーを期待したが、エラーにならなかった")
	}
}

// frontmatterNameは、SKILL.mdのfrontmatterの `name` に一致する。
var frontmatterName = regexp.MustCompile(`(?m)\A---\n(?:.*\n)*?name: (.+)\n(?:.*\n)*?---\n`)

func TestEntryFileFrontmatter(t *testing.T) {
	t.Parallel()

	for _, lang := range langs {
		content := readFile(t, lang, EntryFileName)
		m := frontmatterName.FindStringSubmatch(content)
		if m == nil {
			t.Errorf("%sのSKILL.mdのfrontmatterにnameが無い", lang)
			continue
		}
		// エージェントはnameでスキルを見分け、書き出すディレクトリの名前と揃っている必要がある。
		if m[1] != Name {
			t.Errorf("%sのSKILL.mdのname = %q、期待値 = %q", lang, m[1], Name)
		}
		if !strings.Contains(content, "\ndescription: ") {
			t.Errorf("%sのSKILL.mdのfrontmatterにdescriptionが無い", lang)
		}
	}
}

// markdownLinkは、Markdownのリンクのうち、同じディレクトリのファイルへの相対リンクに一致する。
var markdownLink = regexp.MustCompile(`\]\(([a-z-]+\.md)\)`)

func TestLinksPointToExistingFiles(t *testing.T) {
	t.Parallel()

	for _, lang := range langs {
		names := fileNames(t, lang)
		linked := map[string]bool{}
		for _, name := range names {
			for _, m := range markdownLink.FindAllStringSubmatch(readFile(t, lang, name), -1) {
				linked[m[1]] = true
				if !slices.Contains(names, m[1]) {
					t.Errorf("%s/%sのリンク先 %s が無い", lang, name, m[1])
				}
			}
		}

		// 手順書はSKILL.mdから辿れないと読まれないため、すべての手順書へのリンクがあることを確かめる。
		for _, name := range names {
			if name != EntryFileName && !linked[name] {
				t.Errorf("%s/%sへのリンクがどのファイルにも無い", lang, name)
			}
		}
	}
}

func TestReplaceCommandTable(t *testing.T) {
	t.Parallel()

	content := "# 見出し\n\n" + CommandTableBegin + "\n\n| 古い表 |\n\n" + CommandTableEnd + "\n"
	got, err := ReplaceCommandTable(content, "| 新しい表 |\n")
	if err != nil {
		t.Fatal(err)
	}
	want := "# 見出し\n\n" + CommandTableBegin + "\n\n| 新しい表 |\n\n" + CommandTableEnd + "\n"
	if got != want {
		t.Errorf("ReplaceCommandTable() = %q、期待値 = %q", got, want)
	}

	// 置き換えた結果をもう一度置き換えても変わらない。
	again, err := ReplaceCommandTable(got, "| 新しい表 |\n")
	if err != nil {
		t.Fatal(err)
	}
	if again != got {
		t.Errorf("2回目の置き換えで変わった: %q", again)
	}

	if _, err := ReplaceCommandTable("# 見出し\n", "| 表 |\n"); err == nil {
		t.Error("印の無い中身でエラーを期待したが、エラーにならなかった")
	}
}

func readFile(t *testing.T, lang, name string) string {
	t.Helper()

	files, err := Files(lang)
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

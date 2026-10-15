//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// TestPageは、ページの作成から、取得・検索・リンク・バックリンク、3つの渡し方
// (`--body`・`--md-file`・`--edits`) での更新までを通す。
func TestPage(t *testing.T) {
	topic := strconv.Itoa(sandboxTopicNumber(t))
	word := uniqueWord()
	title := "CLIのe2eのページ " + word
	linkTitle := "CLIのe2eのリンク先 " + word
	link := "[[" + linkTitle + "]]"

	// 作成: 本文のWikiリンクが指す未作成のページは、作成の後に標準エラーへ一覧が出る。
	created := expectExit(t, 0, "", "page", "create", "--topic", topic, "--title", title, "--body", "最初の本文。\n\n"+link+"\n")
	if !strings.Contains(created.stderr, linkTitle) {
		t.Errorf("作成されるリンク先が標準エラーに出ていない: %s", created.stderr)
	}
	var p page
	decode(t, created.stdout, &p)
	number := strconv.Itoa(p.Number)

	// 取得
	var got page
	mustRunJSON(t, &got, "", "page", "get", number)
	if got.Body != "最初の本文。\n\n"+link+"\n" || got.Status != "published" || got.Etag == "" {
		t.Fatalf("取得したページ = %+v", got)
	}

	// 検索: 作成したページだけがタイトルの語に一致する。
	var found list
	mustRunJSON(t, &found, "", "page", "search", word)
	if numbers := pageNumbers(found); !slices.Equal(numbers, []int{p.Number}) {
		t.Errorf("検索の結果のページ番号 = %v、期待値 = [%d]", numbers, p.Number)
	}

	// リンク: Wikiリンクで作られた未公開のページがリンク先に出て、そのバックリンクに作成したページが出る。
	var links list
	mustRunJSON(t, &links, "", "page", "links", number)
	if len(links.Items) != 1 || links.Items[0].Title == nil || *links.Items[0].Title != linkTitle || links.Items[0].Status != "unpublished" {
		t.Fatalf("リンク先 = %+v", links.Items)
	}
	var backlinks list
	mustRunJSON(t, &backlinks, "", "page", "backlinks", strconv.Itoa(links.Items[0].Number))
	if numbers := pageNumbers(backlinks); !slices.Equal(numbers, []int{p.Number}) {
		t.Errorf("バックリンクのページ番号 = %v、期待値 = [%d]", numbers, p.Number)
	}

	// `--body` での更新: 読んだ版で更新でき、古い版での更新は競合 (終了コード6) で拒まれる。
	var updated page
	mustRunJSON(t, &updated, "", "page", "update", number, "--body", "更新した本文。\n\n"+link+"\n", "--if-match", got.Etag)
	if updated.Etag == "" || updated.Etag == got.Etag {
		t.Errorf("更新後の版 = %q、更新前の版 = %q", updated.Etag, got.Etag)
	}
	expectExit(t, 6, "", "page", "update", number, "--body", "古い版からの本文。\n", "--if-match", got.Etag)

	// `--md` の書き出しと `--md-file` の書き戻し: 版とタイトルはfrontmatterから読む。
	md := mustRun(t, "", "page", "get", number, "--md")
	if !strings.Contains(md, "wikino_etag:") {
		t.Fatalf("ページファイルに版が無い:\n%s", md)
	}
	file := filepath.Join(t.TempDir(), "page.md")
	if err := os.WriteFile(file, []byte(strings.Replace(md, "更新した本文。", "ファイルで書き換えた本文。", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	mustRun(t, "", "page", "update", number, "--md-file", file)

	// `--edits` での検索置換: 標準入力から編集を読み、本文の一部だけを置き換える。
	edits := `{"edits": [{"old_str": "ファイルで書き換えた本文。", "new_str": "検索置換した本文。"}]}`
	mustRun(t, edits, "page", "update", number, "--edits", "-")

	mustRunJSON(t, &got, "", "page", "get", number)
	if want := "検索置換した本文。\n\n" + link + "\n"; got.Body != want {
		t.Errorf("更新後の本文 = %q、期待値 = %q", got.Body, want)
	}
	if got.Title == nil || *got.Title != title {
		t.Errorf("更新後のタイトル = %v、期待値 = %q", got.Title, title)
	}
}

// pageNumbersは一覧のページ番号を返す。
func pageNumbers(l list) []int {
	numbers := make([]int, 0, len(l.Items))
	for _, item := range l.Items {
		numbers = append(numbers, item.Number)
	}

	return numbers
}

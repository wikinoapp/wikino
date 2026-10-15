//go:build e2e

package e2e

import (
	"strconv"
	"strings"
	"testing"
)

// TestDraftは、新しいページの下書きの作成から、更新・編集履歴・公開までと、
// 公開したページに作り直した下書きの削除を通す。
func TestDraft(t *testing.T) {
	topic := strconv.Itoa(sandboxTopicNumber(t))
	title := "CLIのe2eの下書き " + uniqueWord()

	// 作成: 新しいページの下書きは、まだ公開していないページとして番号を持つ。
	var created page
	mustRunJSON(t, &created, "", "draft", "create", "--topic", topic, "--title", title, "--body", "下書きの本文。\n")
	number := strconv.Itoa(created.PageNumber)
	if created.PageNumber == 0 || created.Etag == "" {
		t.Fatalf("作成した下書き = %+v", created)
	}

	// 更新: `--body` と `--edits` のどちらでも、更新のたびに版が変わる。
	var updated page
	mustRunJSON(t, &updated, "", "draft", "update", number, "--body", "書き直した下書きの本文。\n", "--if-match", created.Etag)
	expectExit(t, 6, "", "draft", "update", number, "--body", "古い版からの本文。\n", "--if-match", created.Etag)
	edits := `{"edits": [{"old_str": "書き直した", "new_str": "検索置換した"}]}`
	mustRunJSON(t, &updated, edits, "draft", "update", number, "--edits", "-", "--if-match", updated.Etag)
	if updated.Body != "検索置換した下書きの本文。\n" {
		t.Errorf("更新後の下書きの本文 = %q", updated.Body)
	}

	// 編集履歴: 作成と更新でリビジョンが作られ、最新のリビジョンは今の本文を持つ。
	var revisions list
	mustRunJSON(t, &revisions, "", "draft", "revision", "list", number)
	if len(revisions.Items) < 3 {
		t.Fatalf("リビジョンの件数 = %d、期待値 = 3以上 (作成と2回の更新)", len(revisions.Items))
	}
	var revision page
	mustRunJSON(t, &revision, "", "draft", "revision", "get", number, revisions.Items[0].ID)
	if revision.Body != updated.Body {
		t.Errorf("最新のリビジョンの本文 = %q、期待値 = %q", revision.Body, updated.Body)
	}

	// 公開: `--yes` が無い非対話の実行は、公開せずに確認を求めて終了コード9で終える。
	confirm := expectExit(t, 9, "", "draft", "publish", number, "--if-match", updated.Etag)
	if !strings.Contains(confirm.stdout, `"confirmation_required"`) || !strings.Contains(confirm.stdout, "--yes") {
		t.Errorf("確認を求める出力 = %s", confirm.stdout)
	}
	var published page
	mustRunJSON(t, &published, "", "draft", "publish", number, "--if-match", updated.Etag, "--yes")
	if published.Number != created.PageNumber || published.Status != "published" || published.Body != updated.Body {
		t.Errorf("公開したページ = %+v", published)
	}
	expectExit(t, 5, "", "draft", "get", number)

	// 削除: 公開したページに下書きを作り直し、確認を省いて削除する。
	var redraft page
	mustRunJSON(t, &redraft, "", "draft", "create", number, "--body", "捨てる下書きの本文。\n")
	var deleted struct {
		PageNumber int  `json:"page_number"`
		Deleted    bool `json:"deleted"`
	}
	mustRunJSON(t, &deleted, "", "draft", "delete", number, "--if-match", redraft.Etag, "--yes")
	if deleted.PageNumber != created.PageNumber || !deleted.Deleted {
		t.Errorf("削除の出力 = %+v", deleted)
	}
	expectExit(t, 5, "", "draft", "get", number)

	// 下書きの削除はページに影響しない。
	var got page
	mustRunJSON(t, &got, "", "page", "get", number)
	if got.Body != published.Body {
		t.Errorf("下書きの削除後のページの本文 = %q、期待値 = %q", got.Body, published.Body)
	}
}

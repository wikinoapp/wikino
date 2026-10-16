package components_test

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
)

func renderAPITokenScopeCheckboxes(t *testing.T, data components.APITokenScopeCheckboxesData) string {
	t.Helper()

	ctx := i18n.SetLocale(context.Background(), "ja")
	var buf bytes.Buffer
	if err := components.APITokenScopeCheckboxes(data).Render(ctx, &buf); err != nil {
		t.Fatalf("Render()のエラー = %v", err)
	}
	return buf.String()
}

// inputTagはidの付いたinput要素の開始タグを返す
func inputTag(t *testing.T, html, id string) string {
	t.Helper()

	tag := regexp.MustCompile(`<input[^>]*id="` + regexp.QuoteMeta(id) + `"[^>]*>`).FindString(html)
	if tag == "" {
		t.Fatalf("id=%qのinput要素が無い", id)
	}
	return tag
}

// noteTagはidの付いた注記の開始タグを返す
func noteTag(t *testing.T, html, id string) string {
	t.Helper()

	tag := regexp.MustCompile(`<p[^>]*id="` + regexp.QuoteMeta(id) + `"[^>]*>`).FindString(html)
	if tag == "" {
		t.Fatalf("id=%qの注記が無い", id)
	}
	return tag
}

func TestNewAPITokenScopeOptions_含む関係を選択肢の中で引く(t *testing.T) {
	t.Parallel()

	options := components.NewAPITokenScopeOptions(
		[]model.Scope{model.ScopePageRead, model.ScopeDraftPageRead, model.ScopeDraftPageWrite, model.ScopeDraftPageDelete},
		[]model.Scope{model.ScopeDraftPageDelete},
		nil,
	)

	byScope := make(map[model.Scope]components.APITokenScopeOption, len(options))
	for _, o := range options {
		byScope[o.Scope] = o
	}

	read := byScope[model.ScopeDraftPageRead]
	wantImpliedBy := []components.APITokenScopeImplier{
		{Scope: model.ScopeDraftPageWrite, Checked: false},
		{Scope: model.ScopeDraftPageDelete, Checked: true},
	}
	if len(read.ImpliedBy) != len(wantImpliedBy) {
		t.Fatalf("draft_page:readのImpliedBy = %v、期待値 = %v", read.ImpliedBy, wantImpliedBy)
	}
	for i, want := range wantImpliedBy {
		if read.ImpliedBy[i] != want {
			t.Errorf("draft_page:readのImpliedBy[%d] = %v、期待値 = %v", i, read.ImpliedBy[i], want)
		}
	}
	if !read.Implied() {
		t.Error("draft_page:deleteを選んでいるのに、draft_page:readが含まれていない")
	}
	if got := byScope[model.ScopeDraftPageWrite].Implies; len(got) != 1 || got[0] != model.ScopeDraftPageRead {
		t.Errorf("draft_page:writeのImplies = %v、期待値 = [draft_page:read]", got)
	}

	// page:writeが選択肢に無いため、page:readはどれにも含まれない
	if got := byScope[model.ScopePageRead].ImpliedBy; len(got) != 0 {
		t.Errorf("page:readのImpliedBy = %v、期待値 = 空", got)
	}
}

func TestAPITokenScopeCheckboxes_選んだスコープに含まれるスコープを選択済みかつ無効にして注記する(t *testing.T) {
	t.Parallel()

	html := renderAPITokenScopeCheckboxes(t, components.APITokenScopeCheckboxesData{
		Name: "scopes",
		Options: components.NewAPITokenScopeOptions(
			model.APITokenScopes,
			[]model.Scope{model.ScopeDraftPageWrite},
			[]model.Scope{model.ScopeDraftPageRead},
		),
		ErrorID: "scopes-error",
	})

	read := inputTag(t, html, "scopes_draft_page_read")
	for _, want := range []string{
		" checked",
		" disabled",
		`aria-describedby="scopes-error scopes_draft_page_read-note scopes_draft_page_read-implied-by-draft_page_write"`,
	} {
		if !strings.Contains(read, want) {
			t.Errorf("draft_page:readのチェックボックスに%qが無い: %s", want, read)
		}
	}

	shown := noteTag(t, html, "scopes_draft_page_read-implied-by-draft_page_write")
	if strings.Contains(shown, " hidden") {
		t.Errorf("選んだスコープに含まれることの注記が隠れている: %s", shown)
	}
	if !strings.Contains(html, "「下書きの作成・更新」に含まれます") {
		t.Error("含むスコープの表示名を添えた注記が無い")
	}
	hidden := noteTag(t, html, "scopes_draft_page_read-implied-by-draft_page_delete")
	if !strings.Contains(hidden, " hidden") {
		t.Errorf("選んでいないスコープに含まれることの注記が隠れていない: %s", hidden)
	}

	write := inputTag(t, html, "scopes_draft_page_write")
	if !strings.Contains(write, `data-api-token-scope-implies="draft_page:read"`) {
		t.Errorf("含むスコープのチェックボックスに含まれるスコープのdata属性が無い: %s", write)
	}
	if strings.Contains(write, " disabled") {
		t.Errorf("選んだスコープのチェックボックスが無効になっている: %s", write)
	}

	// 含むスコープを選んでいないため、page:readはそのまま選べる
	pageRead := inputTag(t, html, "scopes_page_read")
	for _, unwanted := range []string{" checked", " disabled", "implied-by"} {
		if strings.Contains(pageRead, unwanted) {
			t.Errorf("page:readのチェックボックスに%qがある: %s", unwanted, pageRead)
		}
	}
}

func TestAPITokenScopeCheckboxes_含むスコープが選択肢に無ければ含まれる関係を描画しない(t *testing.T) {
	t.Parallel()

	// OAuthの同意画面で、アプリがdraft_page:readだけを求めたとき
	html := renderAPITokenScopeCheckboxes(t, components.APITokenScopeCheckboxesData{
		Name:    "granted_scopes",
		Options: components.NewAPITokenScopeOptions([]model.Scope{model.ScopeDraftPageRead}, []model.Scope{model.ScopeDraftPageRead}, nil),
	})

	read := inputTag(t, html, "granted_scopes_draft_page_read")
	if strings.Contains(read, " disabled") || strings.Contains(read, "aria-describedby") {
		t.Errorf("含むスコープが無いのに、含まれる状態で描画している: %s", read)
	}
	if strings.Contains(html, "data-api-token-scope-implied-note") {
		t.Error("含むスコープが無いのに、含まれることの注記がある")
	}
}

func TestAPITokenScopeCheckboxes_無効にしたチェックボックスにはフォーカスを当てない(t *testing.T) {
	t.Parallel()

	html := renderAPITokenScopeCheckboxes(t, components.APITokenScopeCheckboxesData{
		Name: "granted_scopes",
		Options: components.NewAPITokenScopeOptions(
			[]model.Scope{model.ScopeDraftPageRead, model.ScopeDraftPageWrite},
			[]model.Scope{model.ScopeDraftPageWrite},
			nil,
		),
		Autofocus: true,
	})

	if strings.Contains(inputTag(t, html, "granted_scopes_draft_page_read"), " autofocus") {
		t.Error("無効にしたチェックボックスにautofocusが付いている")
	}
	if !strings.Contains(inputTag(t, html, "granted_scopes_draft_page_write"), " autofocus") {
		t.Error("最初の無効でないチェックボックスにautofocusが付いていない")
	}
}

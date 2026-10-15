package personal_access_token_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestNew_発行フォームを表示する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "pat-new"
	m := setupPATMember(t, tx, identifier, model.SpaceRoleViewer, true)

	req := newRequest(t, http.MethodGet, "/s/"+identifier+"/settings/personal_access_tokens/new", identifier, m.userID, nil)
	rr := httptest.NewRecorder()
	setupHandler(t, q).New(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Cache-Control"); got != "private, no-cache" {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, "private, no-cache")
	}

	body := rr.Body.String()
	for _, want := range []string{
		`action="/s/` + identifier + `/settings/personal_access_tokens"`,
		`name="csrf_token"`,
		`name="name"`,
		// 付与できるスコープだけを選択肢に出す。
		`value="topic:read"`,
		`value="page:read"`,
		`value="page:write"`,
		`value="draft_page:read"`,
		`value="draft_page:write"`,
		`value="draft_page:delete"`,
		`value="attachment:read"`,
		// 有効期限は選択肢から選び、既定は30日にする。
		`<option value="7">`,
		`<option value="30" selected>`,
		`<option value="90">`,
		`<option value="365">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}
	for _, notWant := range []string{`value="space:admin"`, `value="personal_access_token:write"`} {
		if strings.Contains(body, notWant) {
			t.Errorf("レスポンスに%qが含まれている", notWant)
		}
	}

	// 必須項目のラベルの中に必須のバッジを出す
	for _, tc := range []struct {
		open  string
		label string
		close string
	}{
		{open: `<label class="label" for="name">`, label: "名前", close: "</label>"},
		{open: `<legend class="label">`, label: "スコープ", close: "</legend>"},
		{open: `<label class="label" for="expiration_days">`, label: "有効期限", close: "</label>"},
	} {
		pattern := regexp.QuoteMeta(tc.open) + `\s*` + regexp.QuoteMeta(tc.label) + `\s*<span class="badge" data-variant="outline">\s*必須\s*</span>\s*` + regexp.QuoteMeta(tc.close)
		if !regexp.MustCompile(pattern).MatchString(body) {
			t.Errorf("%sのラベルの中に必須のバッジが無い", tc.label)
		}
	}
	// バッジを出した項目は、入力欄にもrequiredを付けて支援技術とブラウザに必須を伝える。
	// スコープはチェックボックスのためrequiredを付けない (すべてのチェックを求めてしまう)。
	for _, id := range []string{"name", "expiration_days"} {
		if !regexp.MustCompile(`id="` + id + `"[^>]*required`).MatchString(body) {
			t.Errorf("%sにrequiredが付いていない", id)
		}
	}

	crumb := breadcrumb(t, body)
	if !strings.Contains(crumb, `href="/s/`+identifier+`/settings/personal_access_tokens"`) {
		t.Error("パンくずに一覧へのリンクが含まれていない")
	}
}

func TestNew_今の権限では効かないスコープにだけ注記を添える(t *testing.T) {
	t.Parallel()

	// ineffectiveOfは注記を添えるスコープ。スペース閲覧者は読み取りのスコープだけを持つが、
	// トピック編集者のロールがあればページと下書きのスコープも持つ
	tests := []struct {
		name          string
		topicRole     model.TopicRole
		ineffectiveOf []string
	}{
		{
			name:          "スペースのロールで持たないスコープに注記を添える",
			ineffectiveOf: []string{"page:write", "draft_page:read", "draft_page:write", "draft_page:delete"},
		},
		{
			name:      "トピックのロールで持つスコープには注記を添えない",
			topicRole: model.TopicRoleEditor,
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			identifier := "pat-new-ineffective-" + strconv.Itoa(i)
			m := setupPATMember(t, tx, identifier, model.SpaceRoleViewer, true)
			if tt.topicRole != "" {
				topicID := testutil.NewTopicBuilder(t, tx).WithSpaceID(m.spaceID).WithNumber(1).Build()
				testutil.NewTopicMemberBuilder(t, tx).WithSpaceID(m.spaceID).WithTopicID(topicID).WithSpaceMemberID(m.spaceMemberID).WithRole(tt.topicRole).Build()
			}

			req := newRequest(t, http.MethodGet, "/s/"+identifier+"/settings/personal_access_tokens/new", identifier, m.userID, nil)
			rr := httptest.NewRecorder()
			setupHandler(t, testutil.QueriesWithTx(tx)).New(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
			}
			body := rr.Body.String()
			if n := strings.Count(body, "今の権限では効きません"); n != len(tt.ineffectiveOf) {
				t.Errorf("注記の数 = %d、期待値 = %d", n, len(tt.ineffectiveOf))
			}
			// 各選択肢の範囲 (そのチェックボックスから次のチェックボックスまで) に注記があるかを見る
			for _, scope := range model.APITokenScopes {
				option := body[strings.Index(body, `value="`+string(scope)+`"`):]
				if next := strings.Index(option[1:], `type="checkbox"`); next >= 0 {
					option = option[:next+1]
				}
				got := strings.Contains(option, "今の権限では効きません")
				if want := slices.Contains(tt.ineffectiveOf, string(scope)); got != want {
					t.Errorf("%sの注記 = %v、期待値 = %v", scope, got, want)
				}
			}
		})
	}
}
func TestNew_発行の権限が無ければ404が返る(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		identifier  string
		role        model.SpaceRole
		flagEnabled bool
	}{
		{name: "フィーチャーフラグが無効", identifier: "pat-new-noflag", role: model.SpaceRoleAdmin},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			m := setupPATMember(t, tx, tt.identifier, tt.role, tt.flagEnabled)

			req := newRequest(t, http.MethodGet, "/s/"+tt.identifier+"/settings/personal_access_tokens/new", tt.identifier, m.userID, nil)
			rr := httptest.NewRecorder()
			setupHandler(t, q).New(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
			}
		})
	}
}

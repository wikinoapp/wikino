package oauth_application_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestNew_登録フォームを表示する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "oauth-app-new"
	m := setupAppMember(t, tx, identifier, "登録するメンバー", model.SpaceRoleAdmin, true)

	req := newRequest(t, http.MethodGet, "/s/"+identifier+"/settings/oauth_applications/new", identifier, m.userID, nil)
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
		"<title>アプリの登録 | アプリのスペース</title>",
		`action="/s/` + identifier + `/settings/oauth_applications"`,
		`name="csrf_token"`,
		`id="redirect_uris"`,
		`value="confidential"`,
		`value="public"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}
	// 初めて開いたときはconfidentialを選んでおく
	if !regexp.MustCompile(`id="client_type_confidential"[^>]*checked`).MatchString(body) {
		t.Error("confidentialが選ばれていない")
	}

	// 必須項目のラベルの中に、文言を括弧で添える代わりに必須のバッジを出す
	for _, tc := range []struct {
		open  string
		label string
		close string
	}{
		{open: `<label class="label" for="name">`, label: "名前", close: "</label>"},
		{open: `<label class="label" for="redirect_uris">`, label: "リダイレクトURI", close: "</label>"},
		{open: `<legend class="label">`, label: "クライアントの種別", close: "</legend>"},
	} {
		pattern := regexp.QuoteMeta(tc.open) + `\s*` + regexp.QuoteMeta(tc.label) + `\s*<span class="badge" data-variant="outline">\s*必須\s*</span>\s*` + regexp.QuoteMeta(tc.close)
		if !regexp.MustCompile(pattern).MatchString(body) {
			t.Errorf("%sのラベルの中に必須のバッジが無い", tc.label)
		}
	}
	// バッジを出した項目は、入力欄にもrequiredを付けて支援技術とブラウザに必須を伝える
	for _, id := range []string{"name", "redirect_uris", "client_type_confidential", "client_type_public"} {
		if !regexp.MustCompile(`id="` + id + `"[^>]*required`).MatchString(body) {
			t.Errorf("%sにrequiredが付いていない", id)
		}
	}
	if strings.Contains(body, "(必須)") {
		t.Error("ラベルに「(必須)」の文言が残っている")
	}

	// 各項目の補足は、入力欄からaria-describedbyで辿れる
	for _, tc := range []struct {
		inputID string
		hintID  string
	}{
		{inputID: "name", hintID: "name-hint"},
		{inputID: "redirect_uris", hintID: "redirect_uris-hint"},
		{inputID: "client_type_confidential", hintID: "client_type_confidential-hint"},
		{inputID: "client_type_public", hintID: "client_type_public-hint"},
	} {
		if !regexp.MustCompile(`id="` + tc.inputID + `"[^>]*aria-describedby="` + tc.hintID + `"`).MatchString(body) {
			t.Errorf("%sのaria-describedbyが補足 (%s) を指していない", tc.inputID, tc.hintID)
		}
		if !strings.Contains(body, `id="`+tc.hintID+`"`) {
			t.Errorf("補足の要素 (%s) が無い", tc.hintID)
		}
	}
}

func TestNew_登録の権限が無ければ404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "oauth-app-new-readonly"
	m := setupAppMember(t, tx, identifier, "編集者のメンバー", model.SpaceRoleEditor, true)

	req := newRequest(t, http.MethodGet, "/s/"+identifier+"/settings/oauth_applications/new", identifier, m.userID, nil)
	rr := httptest.NewRecorder()
	setupHandler(t, q).New(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
	}
}

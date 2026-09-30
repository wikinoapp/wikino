package personal_access_token_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestIndex_自分のトークンの一覧を表示する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "pat-index"
	m := setupPATMember(t, tx, identifier, model.SpaceRoleAdmin, true)
	other := setupPATMember(t, tx, "pat-index-other", model.SpaceRoleAdmin, true)

	lastUsedAt := time.Now().Add(-time.Hour)
	activeID := testutil.NewPersonalAccessTokenBuilder(t, tx).
		WithSpaceID(m.spaceID).WithSpaceMemberID(m.spaceMemberID).
		WithName("自宅のCLI").WithTokenDigest("pat_index_active").
		WithScopes([]model.Scope{model.ScopeTopicRead, model.ScopePageWrite}).
		WithLastUsedAt(lastUsedAt).
		Build()
	testutil.NewPersonalAccessTokenBuilder(t, tx).
		WithSpaceID(m.spaceID).WithSpaceMemberID(m.spaceMemberID).
		WithName("古いスクリプト").WithTokenDigest("pat_index_expired").
		WithExpiresAt(time.Now().Add(-time.Hour)).
		Build()
	testutil.NewPersonalAccessTokenBuilder(t, tx).
		WithSpaceID(m.spaceID).WithSpaceMemberID(m.spaceMemberID).
		WithName("失効したトークン").WithTokenDigest("pat_index_revoked").
		WithRevokedAt(time.Now()).
		Build()
	testutil.NewPersonalAccessTokenBuilder(t, tx).
		WithSpaceID(other.spaceID).WithSpaceMemberID(other.spaceMemberID).
		WithName("他人のトークン").WithTokenDigest("pat_index_other").
		Build()

	req := newRequest(t, http.MethodGet, "/s/"+identifier+"/settings/personal_access_tokens", identifier, m.userID, nil)
	rr := httptest.NewRecorder()
	setupHandler(t, q).Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Cache-Control"); got != "private, no-cache" {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, "private, no-cache")
	}

	body := rr.Body.String()
	for _, want := range []string{
		"<title>個人アクセストークン | トークンのスペース</title>",
		"自宅のCLI",
		"…abcd",
		"トピックの読み取り",
		"ページの作成・更新",
		"古いスクリプト",
		"期限切れ",
		"未使用",
		`href="/s/` + identifier + `/settings/personal_access_tokens/new"`,
		`action="/s/` + identifier + `/settings/personal_access_tokens/` + string(activeID) + `"`,
		`name="_method" value="DELETE"`,
		"トークンを失効する",
		// 同じ文言の失効ボタンを見分けられるよう、ボタンの名前にトークン名を連ねる
		`aria-labelledby="personal-access-token-` + string(activeID) + `-revoke-label personal-access-token-` + string(activeID) + `-name"`,
		`id="personal-access-token-` + string(activeID) + `-name"`,
		`id="personal-access-token-` + string(activeID) + `-revoke-label"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}
	for _, notWant := range []string{"失効したトークン", "他人のトークン"} {
		if strings.Contains(body, notWant) {
			t.Errorf("レスポンスに%qが含まれている", notWant)
		}
	}

	crumb := breadcrumb(t, body)
	for _, want := range []string{`href="/s/` + identifier + `/settings"`, `aria-current="page"`} {
		if !strings.Contains(crumb, want) {
			t.Errorf("パンくずに%qが含まれていない", want)
		}
	}
}

func TestIndex_閲覧者にも一覧と失効ボタンを表示する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "pat-index-delete-only"
	m := setupPATMember(t, tx, identifier, model.SpaceRoleViewer, true)
	tokenID := testutil.NewPersonalAccessTokenBuilder(t, tx).
		WithSpaceID(m.spaceID).WithSpaceMemberID(m.spaceMemberID).
		WithTokenDigest("pat_index_delete_only").Build()

	req := newRequest(t, http.MethodGet, "/s/"+identifier+"/settings/personal_access_tokens", identifier, m.userID, nil)
	rr := httptest.NewRecorder()
	setupHandler(t, q).Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), `action="/s/`+identifier+`/settings/personal_access_tokens/`+string(tokenID)+`"`) {
		t.Error("失効フォームが含まれていない")
	}
}

func TestIndex_開けない場合は404が返る(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		identifier  string
		role        model.SpaceRole
		flagEnabled bool
	}{
		{name: "フィーチャーフラグが無効", identifier: "pat-index-noflag", role: model.SpaceRoleAdmin},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			m := setupPATMember(t, tx, tt.identifier, tt.role, tt.flagEnabled)

			req := newRequest(t, http.MethodGet, "/s/"+tt.identifier+"/settings/personal_access_tokens", tt.identifier, m.userID, nil)
			rr := httptest.NewRecorder()
			setupHandler(t, q).Index(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
			}
		})
	}
}

func TestIndex_未ログインならログイン画面へリダイレクトする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "pat-index-anon"
	setupPATMember(t, tx, identifier, model.SpaceRoleAdmin, true)

	req := newRequest(t, http.MethodGet, "/s/"+identifier+"/settings/personal_access_tokens", identifier, "", nil)
	rr := httptest.NewRecorder()
	setupHandler(t, q).Index(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusFound)
	}
	if location := rr.Header().Get("Location"); location != "/sign_in" {
		t.Errorf("Location = %q、期待値 = %q", location, "/sign_in")
	}
}

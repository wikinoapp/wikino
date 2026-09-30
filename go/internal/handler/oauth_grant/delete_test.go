package oauth_grant_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestDelete_自分の連携を解除して一覧へ戻す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "grant-delete"
	m := setupGrantMember(t, tx, identifier, []model.Scope{model.ScopeOAuthGrantDelete}, true)
	grantID := m.buildGrant(t, tx, m.spaceMemberID, "自宅のCLI", "grant-delete-cli", []model.Scope{model.ScopePageRead})
	testutil.NewOAuthAccessTokenBuilder(t, tx).
		WithOAuthGrantID(grantID).WithSpaceID(m.spaceID).WithTokenDigest("grant_delete_access_digest").Build()

	req := newRequest(t, http.MethodDelete, "/s/"+identifier+"/settings/oauth_grants/"+string(grantID), identifier, m.userID)
	rr := httptest.NewRecorder()
	handler := setupHandler(t, q)
	handler.Delete(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusSeeOther)
	}
	if location, want := rr.Header().Get("Location"), "/s/"+identifier+"/settings/oauth_grants"; location != want {
		t.Errorf("Location = %q、期待値 = %q", location, want)
	}
	if cookie := rr.Header().Get("Set-Cookie"); !strings.Contains(cookie, "flash") {
		t.Errorf("Set-Cookie = %q、フラッシュのCookieを期待", cookie)
	}

	ctx := context.Background()
	grants, err := repository.NewOAuthGrantRepository(q).ListUnrevokedBySpaceMember(ctx, m.spaceID, m.spaceMemberID)
	if err != nil {
		t.Fatalf("ListUnrevokedBySpaceMember()のエラー = %v", err)
	}
	if len(grants) != 0 {
		t.Errorf("失効していない許可の件数 = %d、期待値 = 0", len(grants))
	}
	accessToken, err := repository.NewOAuthAccessTokenRepository(q).FindByTokenDigest(ctx, "grant_delete_access_digest")
	if err != nil {
		t.Fatalf("FindByTokenDigest()のエラー = %v", err)
	}
	if !accessToken.IsRevoked() {
		t.Error("許可に属するアクセストークンが失効していない")
	}

	// :deleteだけのメンバーも、解除後のリダイレクト先を表示できる。
	indexReq := newRequest(t, http.MethodGet, "/s/"+identifier+"/settings/oauth_grants", identifier, m.userID)
	indexRR := httptest.NewRecorder()
	handler.Index(indexRR, indexReq)
	if indexRR.Code != http.StatusOK {
		t.Errorf("解除後の一覧のステータスコード = %d、期待値 = %d", indexRR.Code, http.StatusOK)
	}
}

func TestDelete_解除できない場合は404が返る(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		identifier  string
		scopes      []model.Scope
		flagEnabled bool
		// otherMemberが真なら、同じスペースの他のメンバーの連携を解除しようとする
		otherMember bool
	}{
		{name: "oauth_grant:deleteを持たない", identifier: "grant-delete-noscope", scopes: []model.Scope{model.ScopeOAuthGrantWrite}, flagEnabled: true},
		{name: "フィーチャーフラグが無効", identifier: "grant-delete-noflag", scopes: []model.Scope{model.ScopeSpaceAdmin}},
		{name: "他のメンバーの連携", identifier: "grant-delete-other", scopes: []model.Scope{model.ScopeSpaceAdmin}, flagEnabled: true, otherMember: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			m := setupGrantMember(t, tx, tt.identifier, tt.scopes, tt.flagEnabled)

			spaceMemberID := m.spaceMemberID
			if tt.otherMember {
				otherUserID := testutil.NewUserBuilder(t, tx).WithEmail(tt.identifier + "-other@example.com").WithAtname("grant_delete_other_owner").Build()
				spaceMemberID = testutil.NewSpaceMemberBuilder(t, tx).
					WithSpaceID(m.spaceID).WithUserID(otherUserID).WithScopes([]model.Scope{model.ScopeSpaceAdmin}).Build()
			}
			grantID := m.buildGrant(t, tx, spaceMemberID, "自宅のCLI", tt.identifier+"-cli", []model.Scope{model.ScopePageRead})

			req := newRequest(t, http.MethodDelete, "/s/"+tt.identifier+"/settings/oauth_grants/"+string(grantID), tt.identifier, m.userID)
			rr := httptest.NewRecorder()
			setupHandler(t, q).Delete(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
			}

			grants, err := repository.NewOAuthGrantRepository(q).ListUnrevokedBySpaceMember(context.Background(), m.spaceID, spaceMemberID)
			if err != nil {
				t.Fatalf("ListUnrevokedBySpaceMember()のエラー = %v", err)
			}
			if len(grants) != 1 {
				t.Errorf("失効していない許可の件数 = %d、期待値 = 1", len(grants))
			}
		})
	}
}

func TestDelete_未ログインならログイン画面へリダイレクトする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "grant-delete-anon"
	setupGrantMember(t, tx, identifier, []model.Scope{model.ScopeSpaceAdmin}, true)

	req := newRequest(t, http.MethodDelete, "/s/"+identifier+"/settings/oauth_grants/00000000-0000-0000-0000-000000000000", identifier, "")
	rr := httptest.NewRecorder()
	setupHandler(t, q).Delete(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusFound)
	}
	if location := rr.Header().Get("Location"); location != "/sign_in" {
		t.Errorf("Location = %q、期待値 = %q", location, "/sign_in")
	}
}

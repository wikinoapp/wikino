package personal_access_token_test

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

func TestDelete_自分のトークンを失効して一覧へ戻す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "pat-delete"
	m := setupPATMember(t, tx, identifier, []model.Scope{model.ScopePersonalAccessTokenDelete}, true)
	tokenID := testutil.NewPersonalAccessTokenBuilder(t, tx).
		WithSpaceID(m.spaceID).WithSpaceMemberID(m.spaceMemberID).
		WithName("自宅のCLI").WithTokenDigest("pat_delete").
		Build()

	req := newRequest(t, http.MethodDelete, "/s/"+identifier+"/settings/personal_access_tokens/"+string(tokenID), identifier, m.userID, nil)
	rr := httptest.NewRecorder()
	handler := setupHandler(t, q)
	handler.Delete(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusSeeOther)
	}
	if location, want := rr.Header().Get("Location"), "/s/"+identifier+"/settings/personal_access_tokens"; location != want {
		t.Errorf("Location = %q、期待値 = %q", location, want)
	}
	if cookie := rr.Header().Get("Set-Cookie"); !strings.Contains(cookie, "flash") {
		t.Errorf("Set-Cookie = %q、フラッシュのCookieを期待", cookie)
	}

	tokens, err := repository.NewPersonalAccessTokenRepository(q).ListUnrevokedBySpaceMember(context.Background(), m.spaceID, m.spaceMemberID)
	if err != nil {
		t.Fatalf("ListUnrevokedBySpaceMember()のエラー = %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("失効していないトークンの件数 = %d、期待値 = 0", len(tokens))
	}

	// :deleteだけのメンバーも、失効後のリダイレクト先を表示できる。
	indexReq := newRequest(t, http.MethodGet, "/s/"+identifier+"/settings/personal_access_tokens", identifier, m.userID, nil)
	indexRR := httptest.NewRecorder()
	handler.Index(indexRR, indexReq)
	if indexRR.Code != http.StatusOK {
		t.Errorf("失効後の一覧のステータスコード = %d、期待値 = %d", indexRR.Code, http.StatusOK)
	}
}

func TestDelete_失効できない場合は404が返る(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		identifier  string
		scopes      []model.Scope
		flagEnabled bool
		// otherMemberが真なら、同じスペースの他のメンバーのトークンを失効しようとする
		otherMember bool
	}{
		{name: "personal_access_token:deleteを持たない", identifier: "pat-delete-noscope", scopes: []model.Scope{model.ScopePersonalAccessTokenWrite}, flagEnabled: true},
		{name: "フィーチャーフラグが無効", identifier: "pat-delete-noflag", scopes: []model.Scope{model.ScopeSpaceAdmin}},
		{name: "他のメンバーのトークン", identifier: "pat-delete-other", scopes: []model.Scope{model.ScopeSpaceAdmin}, flagEnabled: true, otherMember: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			m := setupPATMember(t, tx, tt.identifier, tt.scopes, tt.flagEnabled)

			spaceMemberID := m.spaceMemberID
			if tt.otherMember {
				otherUserID := testutil.NewUserBuilder(t, tx).WithEmail(tt.identifier + "-other@example.com").Build()
				spaceMemberID = testutil.NewSpaceMemberBuilder(t, tx).
					WithSpaceID(m.spaceID).WithUserID(otherUserID).WithScopes([]model.Scope{model.ScopeSpaceAdmin}).Build()
			}
			tokenID := testutil.NewPersonalAccessTokenBuilder(t, tx).
				WithSpaceID(m.spaceID).WithSpaceMemberID(spaceMemberID).WithTokenDigest(tt.identifier).
				Build()

			req := newRequest(t, http.MethodDelete, "/s/"+tt.identifier+"/settings/personal_access_tokens/"+string(tokenID), tt.identifier, m.userID, nil)
			rr := httptest.NewRecorder()
			setupHandler(t, q).Delete(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
			}

			tokens, err := repository.NewPersonalAccessTokenRepository(q).ListUnrevokedBySpaceMember(context.Background(), m.spaceID, spaceMemberID)
			if err != nil {
				t.Fatalf("ListUnrevokedBySpaceMember()のエラー = %v", err)
			}
			if len(tokens) != 1 {
				t.Errorf("失効していないトークンの件数 = %d、期待値 = 1", len(tokens))
			}
		})
	}
}

func TestDelete_未ログインならログイン画面へリダイレクトする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "pat-delete-anon"
	setupPATMember(t, tx, identifier, []model.Scope{model.ScopeSpaceAdmin}, true)

	req := newRequest(t, http.MethodDelete, "/s/"+identifier+"/settings/personal_access_tokens/00000000-0000-0000-0000-000000000000", identifier, "", nil)
	rr := httptest.NewRecorder()
	setupHandler(t, q).Delete(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusFound)
	}
	if location := rr.Header().Get("Location"); location != "/sign_in" {
		t.Errorf("Location = %q、期待値 = %q", location, "/sign_in")
	}
}

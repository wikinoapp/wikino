package oauth_application_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestDelete_アプリを削除して許可を失効し一覧へリダイレクトする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "oauth-app-delete"
	m := setupAppMember(t, tx, identifier, "削除するメンバー", []model.Scope{model.ScopeOAuthApplicationDelete}, true)
	appID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(m.spaceID).
		WithClientID("oauth-app-delete-client").
		Build()
	grantID := testutil.NewOAuthGrantBuilder(t, tx).
		WithOAuthApplicationID(appID).
		WithSpaceID(m.spaceID).
		WithSpaceMemberID(m.spaceMemberID).
		Build()

	path := "/s/" + identifier + "/settings/oauth_applications/" + string(appID)
	req := newRequest(t, http.MethodDelete, path, identifier, m.userID, nil)
	rr := httptest.NewRecorder()
	setupHandler(t, q).Delete(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusSeeOther)
	}
	if want := "/s/" + identifier + "/settings/oauth_applications"; rr.Header().Get("Location") != want {
		t.Errorf("Location = %q、期待値 = %q", rr.Header().Get("Location"), want)
	}

	ctx := context.Background()
	if app, err := repository.NewOAuthApplicationRepository(q).FindByIDAndSpaceID(ctx, appID, m.spaceID); err != nil || app != nil {
		t.Errorf("削除後のFindByIDAndSpaceID() = (%v, %v)、期待値 = (nil, nil)", app, err)
	}
	grant, err := repository.NewOAuthGrantRepository(q).FindByID(ctx, grantID, m.spaceID)
	if err != nil {
		t.Fatalf("FindByID()のエラー = %v", err)
	}
	if grant.RevokedAt == nil {
		t.Error("削除したアプリの許可が失効していない")
	}
}

func TestDelete_削除できないメンバーとアプリには404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "oauth-app-delete-denied"
	deleter := setupAppMember(t, tx, identifier, "削除するメンバー", []model.Scope{model.ScopeOAuthApplicationDelete}, true)
	writer := setupAppMember(t, tx, identifier+"-writer", "編集だけのメンバー", []model.Scope{model.ScopeOAuthApplicationWrite}, true)
	writerAppID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(writer.spaceID).
		WithClientID("oauth-app-delete-writer-client").
		Build()

	// サブテストはフィクスチャのトランザクションを共有するため、並行に走らせない。
	for _, tt := range []struct {
		name       string
		identifier string
		userID     model.UserID
	}{
		{name: "oauth_application:writeだけを持つ", identifier: identifier + "-writer", userID: writer.userID},
		{name: "別のスペースのアプリ", identifier: identifier, userID: deleter.userID},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := "/s/" + tt.identifier + "/settings/oauth_applications/" + string(writerAppID)
			req := newRequest(t, http.MethodDelete, path, tt.identifier, tt.userID, nil)
			rr := httptest.NewRecorder()
			setupHandler(t, q).Delete(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
			}
		})
	}

	app, err := repository.NewOAuthApplicationRepository(q).FindByIDAndSpaceID(context.Background(), writerAppID, writer.spaceID)
	if err != nil || app == nil {
		t.Errorf("FindByIDAndSpaceID() = (%v, %v)、期待値は削除されていないアプリ", app, err)
	}
}

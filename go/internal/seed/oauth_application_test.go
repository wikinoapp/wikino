package seed

import (
	"context"
	"slices"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestRegisterOfficialOAuthApplications(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	if err := registerOfficialOAuthApplications(ctx, tx); err != nil {
		t.Fatalf("公式クライアントの登録に失敗: %v", err)
	}

	app, err := repository.NewOAuthApplicationRepository(testutil.QueriesWithTx(tx)).FindByClientID(ctx, model.OfficialCLIClientID)
	if err != nil {
		t.Fatalf("FindByClientID()のエラー = %v", err)
	}
	if app == nil {
		t.Fatalf("client_id = %qのアプリが登録されていない", model.OfficialCLIClientID)
	}
	if !app.IsOfficialClient() || app.IsConfidential() || app.CreatedSpaceMemberID != nil {
		t.Errorf("アプリ = %+v、作成者の無い公式のpublicクライアントを期待", app)
	}
	if want := []string{"http://127.0.0.1/callback", "http://[::1]/callback"}; !slices.Equal(app.RedirectURIs, want) {
		t.Errorf("RedirectURIs = %v、期待値 = %v", app.RedirectURIs, want)
	}
	for _, uri := range []string{"http://127.0.0.1:53123/callback", "http://[::1]:53123/callback"} {
		if !app.AcceptsRedirectURI(uri) {
			t.Errorf("AcceptsRedirectURI(%q) = false、ループバックのポート番号を問わず受け付けることを期待", uri)
		}
	}
}

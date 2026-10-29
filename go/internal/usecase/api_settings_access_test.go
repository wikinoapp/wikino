package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestAPISettingsAccess_MCPサーバーだけが有効でも公開APIの管理は許可しない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	ctx := context.Background()
	f := setupPATMember(t, tx, "api-settings-mcp-only", model.SpaceRoleAdmin, false)
	testutil.NewFeatureFlagBuilder(t, tx).
		WithUserID(f.userID).
		WithName(string(model.FeatureFlagRemoteMCPServer)).
		Build()

	_, err := newGetPersonalAccessTokensUsecaseForTest(q).Execute(ctx, GetPersonalAccessTokensInput{
		SpaceIdentifier: "api-settings-mcp-only", UserID: f.userID,
	})
	assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)

	_, err = newGetOAuthApplicationsUsecaseForTest(q).Execute(ctx, GetOAuthApplicationsInput{
		SpaceIdentifier: "api-settings-mcp-only", UserID: f.userID,
	})
	assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
}

package viewmodel_test

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestNewOAuthGrants_CIMDのクライアントにはクライアントIDのホスト名を添える(t *testing.T) {
	t.Parallel()

	grants := []*model.OAuthGrant{
		{ID: "grant-cimd", OAuthApplicationID: "app-cimd"},
		{ID: "grant-registered", OAuthApplicationID: "app-registered"},
	}
	apps := map[model.OAuthApplicationID]*model.OAuthApplication{
		"app-cimd":       {ID: "app-cimd", Name: "Claude", ClientID: "https://claude.ai/oauth/metadata", RegistrationMethod: model.OAuthRegistrationMethodClientIDMetadataDocument},
		"app-registered": {ID: "app-registered", Name: "連携するCLI", ClientID: "space-client"},
	}

	vms := viewmodel.NewOAuthGrants(grants, apps)
	if len(vms) != 2 {
		t.Fatalf("len(vms) = %d、期待値 = 2", len(vms))
	}
	if vms[0].ApplicationHost != "claude.ai" {
		t.Errorf("CIMDのクライアントのApplicationHost = %q、期待値 = %q", vms[0].ApplicationHost, "claude.ai")
	}
	if vms[1].ApplicationHost != "" {
		t.Errorf("登録したクライアントのApplicationHost = %q、期待値 = 空", vms[1].ApplicationHost)
	}
}

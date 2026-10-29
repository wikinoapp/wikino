package model

import (
	"testing"
	"time"
)

func TestOAuthApplication_IsAvailableIn(t *testing.T) {
	t.Parallel()

	spaceID := SpaceID("space")

	tests := []struct {
		name               string
		spaceID            *SpaceID
		registrationMethod OAuthRegistrationMethod
		target             SpaceID
		want               bool
	}{
		{name: "スペースのアプリは登録したスペースで使える", spaceID: &spaceID, target: "space", want: true},
		{name: "スペースのアプリは他のスペースで使えない", spaceID: &spaceID, target: "other", want: false},
		{name: "公式クライアントはどのスペースでも使える", target: "other", want: true},
		{name: "CIMDのクライアントはどのスペースでも使える", registrationMethod: OAuthRegistrationMethodClientIDMetadataDocument, target: "other", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			app := &OAuthApplication{SpaceID: tt.spaceID, RegistrationMethod: tt.registrationMethod}
			if got := app.IsAvailableIn(tt.target); got != tt.want {
				t.Errorf("IsAvailableIn() = %v、期待値 = %v", got, tt.want)
			}
		})
	}
}

func TestOAuthApplication_IsOfficialClient(t *testing.T) {
	t.Parallel()

	spaceID := SpaceID("space")

	tests := []struct {
		name string
		app  OAuthApplication
		want bool
	}{
		{name: "スペースに属さない登録したクライアントは公式クライアント", app: OAuthApplication{}, want: true},
		{name: "スペースのアプリは公式クライアントでない", app: OAuthApplication{SpaceID: &spaceID}, want: false},
		{name: "CIMDのクライアントは公式クライアントでない", app: OAuthApplication{RegistrationMethod: OAuthRegistrationMethodClientIDMetadataDocument}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.app.IsOfficialClient(); got != tt.want {
				t.Errorf("IsOfficialClient() = %v、期待値 = %v", got, tt.want)
			}
		})
	}
}

func TestOAuthApplication_IsMetadataFresh(t *testing.T) {
	t.Parallel()

	now := time.Now()
	future := now.Add(time.Minute)

	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{name: "期限より前なら期限内", expiresAt: &future, want: true},
		{name: "期限ちょうどなら期限切れ", expiresAt: &now, want: false},
		{name: "期限が無ければ期限切れとして扱う", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			app := &OAuthApplication{MetadataExpiresAt: tt.expiresAt}
			if got := app.IsMetadataFresh(now); got != tt.want {
				t.Errorf("IsMetadataFresh() = %v、期待値 = %v", got, tt.want)
			}
		})
	}
}

func TestOAuthApplication_ClientIDHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		app  OAuthApplication
		want string
	}{
		{name: "CIMDのクライアントはクライアントIDのホスト名を返す", app: OAuthApplication{ClientID: "https://claude.ai/oauth/metadata", RegistrationMethod: OAuthRegistrationMethodClientIDMetadataDocument}, want: "claude.ai"},
		{name: "登録したクライアントは空を返す", app: OAuthApplication{ClientID: "wikino-cli"}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.app.ClientIDHost(); got != tt.want {
				t.Errorf("ClientIDHost() = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}

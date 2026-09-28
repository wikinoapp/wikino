package model

import "testing"

func TestOAuthApplication_IsAvailableIn(t *testing.T) {
	t.Parallel()

	spaceID := SpaceID("space")

	tests := []struct {
		name    string
		spaceID *SpaceID
		target  SpaceID
		want    bool
	}{
		{name: "スペースのアプリは登録したスペースで使える", spaceID: &spaceID, target: "space", want: true},
		{name: "スペースのアプリは他のスペースで使えない", spaceID: &spaceID, target: "other", want: false},
		{name: "公式クライアントはどのスペースでも使える", target: "other", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			app := &OAuthApplication{SpaceID: tt.spaceID}
			if got := app.IsAvailableIn(tt.target); got != tt.want {
				t.Errorf("IsAvailableIn() = %v、期待値 = %v", got, tt.want)
			}
		})
	}
}

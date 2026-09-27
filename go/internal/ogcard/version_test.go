package ogcard

import (
	"testing"
)

func TestCardVersion(t *testing.T) {
	t.Parallel()

	base := Card{SpaceName: "スペース", TopicName: "トピック", PageTitle: "ページ"}

	t.Run("内容が同じなら同じバージョンになる", func(t *testing.T) {
		t.Parallel()

		same := Card{SpaceName: "スペース", TopicName: "トピック", PageTitle: "ページ"}
		if got, want := same.Version(), base.Version(); got != want {
			t.Errorf("Version() = %q、期待値 = %q", got, want)
		}
	})

	t.Run("バージョンは16文字の16進数になる", func(t *testing.T) {
		t.Parallel()

		v := base.Version()
		if len(v) != 16 {
			t.Errorf("Version()の長さ = %d、期待値 = 16", len(v))
		}
		for _, c := range v {
			if (c < '0' || '9' < c) && (c < 'a' || 'f' < c) {
				t.Errorf("Version() = %q、16進数でない文字を含む", v)
				break
			}
		}
	})

	tests := []struct {
		name string
		card Card
	}{
		{
			name: "スペース名が変わるとバージョンが変わる",
			card: Card{SpaceName: "別のスペース", TopicName: "トピック", PageTitle: "ページ"},
		},
		{
			name: "トピック名が変わるとバージョンが変わる",
			card: Card{SpaceName: "スペース", TopicName: "別のトピック", PageTitle: "ページ"},
		},
		{
			name: "ページタイトルが変わるとバージョンが変わる",
			card: Card{SpaceName: "スペース", TopicName: "トピック", PageTitle: "別のページ"},
		},
		{
			name: "値の境界がずれるとバージョンが変わる",
			card: Card{SpaceName: "スペーストピック", TopicName: "", PageTitle: "ページ"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.card.Version() == base.Version() {
				t.Errorf("Version() = %q、元の内容と異なる値を期待", tt.card.Version())
			}
		})
	}
}

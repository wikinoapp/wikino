package legaldoc

import (
	"slices"
	"testing"
)

func TestLookup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path     string
		wantOK   bool
		wantPage string
	}{
		{path: "/terms", wantOK: true, wantPage: "/s/wikino/pages/41"},
		{path: "/terms/api", wantOK: true, wantPage: "/s/wikino/pages/323"},
		{path: "/privacy", wantOK: true, wantPage: "/s/wikino/pages/42"},
		{path: "/en/terms", wantOK: true, wantPage: "/s/wikino/pages/325"},
		{path: "/en/terms/api", wantOK: true, wantPage: "/s/wikino/pages/326"},
		{path: "/en/privacy", wantOK: true, wantPage: "/s/wikino/pages/328"},
		{path: "/terms/", wantOK: false},
		{path: "/ja/terms", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()

			doc, ok := Lookup(tt.path)
			if ok != tt.wantOK {
				t.Fatalf("Lookup(%q)のok = %v、期待値 = %v", tt.path, ok, tt.wantOK)
			}
			if doc.PagePath != tt.wantPage {
				t.Errorf("Lookup(%q).PagePath = %q、期待値 = %q", tt.path, doc.PagePath, tt.wantPage)
			}
		})
	}
}

func TestPaths(t *testing.T) {
	t.Parallel()

	want := []string{"/terms", "/terms/api", "/privacy", "/en/terms", "/en/terms/api", "/en/privacy"}
	if got := Paths(); !slices.Equal(got, want) {
		t.Errorf("Paths() = %v、期待値 = %v", got, want)
	}
}

func TestPagePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		key    Key
		locale string
		want   string
	}{
		{name: "日本語なら日本語版", key: KeyTerms, locale: "ja", want: "/s/wikino/pages/41"},
		{name: "英語なら英語版", key: KeyTerms, locale: "en", want: "/s/wikino/pages/325"},
		{name: "日本語以外の言語は英語版", key: KeyPrivacyPolicy, locale: "fr", want: "/s/wikino/pages/328"},
		{name: "言語が空なら英語版", key: KeyAPITerms, locale: "", want: "/s/wikino/pages/326"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := PagePath(tt.key, tt.locale); got != tt.want {
				t.Errorf("PagePath(%q, %q) = %q、期待値 = %q", tt.key, tt.locale, got, tt.want)
			}
		})
	}
}

func TestPagePath_一覧表に無い文書ならpanicする(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Error("panicを期待したが、panicしなかった")
		}
	}()
	PagePath(Key("unknown"), LangJa)
}

func TestPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key  Key
		want string
	}{
		{key: KeyTerms, want: "/terms"},
		{key: KeyAPITerms, want: "/terms/api"},
		{key: KeyPrivacyPolicy, want: "/privacy"},
	}
	for _, tt := range tests {
		t.Run(string(tt.key), func(t *testing.T) {
			t.Parallel()

			if got := Path(tt.key); got != tt.want {
				t.Errorf("Path(%q) = %q、期待値 = %q", tt.key, got, tt.want)
			}
		})
	}
}

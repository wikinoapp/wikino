package apietag_test

import (
	"slices"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/apietag"
)

func TestParseIfMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		ifMatch      string
		wantDigests  []string
		wantMatchAny bool
	}{
		{name: "1つのエンティティタグ", ifMatch: `"abc"`, wantDigests: []string{"abc"}},
		{name: "カンマ区切りの複数のエンティティタグ", ifMatch: `"abc", "def"`, wantDigests: []string{"abc", "def"}},
		{name: "*はどの版とも一致する", ifMatch: " * ", wantMatchAny: true},
		{name: "弱いエンティティタグはW/を外したダイジェストで照合する", ifMatch: `W/"abc", "def"`, wantDigests: []string{"abc", "def"}},
		{name: "W/の後ろが引用符で囲まれていなければ除く", ifMatch: `W/abc`, wantDigests: nil},
		{name: "引用符の無いエンティティタグは除く", ifMatch: `abc`, wantDigests: nil},
		{name: "空の値は何とも一致しない", ifMatch: "", wantDigests: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			digests, matchAny := apietag.ParseIfMatch(tt.ifMatch)
			if !slices.Equal(digests, tt.wantDigests) || matchAny != tt.wantMatchAny {
				t.Errorf("ParseIfMatch(%q) = (%v, %v)、期待値 = (%v, %v)", tt.ifMatch, digests, matchAny, tt.wantDigests, tt.wantMatchAny)
			}
		})
	}
}

func TestIfNoneMatchMatches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		ifNoneMatch string
		want        bool
	}{
		{name: "同じエンティティタグは一致する", ifNoneMatch: `"abc"`, want: true},
		{name: "並びのいずれかと一致すれば一致する", ifNoneMatch: `"def", "abc"`, want: true},
		{name: "弱いエンティティタグも一致する", ifNoneMatch: `W/"abc"`, want: true},
		{name: "*は一致する", ifNoneMatch: "*", want: true},
		{name: "異なるエンティティタグは一致しない", ifNoneMatch: `"def"`, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := apietag.IfNoneMatchMatches(tt.ifNoneMatch, `"abc"`); got != tt.want {
				t.Errorf("IfNoneMatchMatches(%q) = %v、期待値 = %v", tt.ifNoneMatch, got, tt.want)
			}
		})
	}
}

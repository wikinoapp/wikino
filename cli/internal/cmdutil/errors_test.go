package cmdutil

import (
	"errors"
	"fmt"
	"testing"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/i18n"
)

func TestNewFlagError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		ja   string
		en   string
	}{
		{
			name: "未知のフラグ",
			args: []string{"--unknown"},
			ja:   "不明なフラグです: --unknown",
			en:   "Unknown flag: --unknown",
		},
		{
			name: "未知の短縮フラグ",
			args: []string{"-z"},
			ja:   "不明なフラグです: -z",
			en:   "Unknown flag: -z",
		},
		{
			name: "組の末尾の未知の短縮フラグ",
			args: []string{"-hz"},
			ja:   "不明なフラグです: -z",
			en:   "Unknown flag: -z",
		},
		{
			name: "組の中の未知の短縮フラグ",
			args: []string{"-hzq"},
			ja:   "不明な短縮フラグです: -z (-zq)",
			en:   "Unknown shorthand flag: -z (-zq)",
		},
		{
			name: "値の欠落",
			args: []string{"--space"},
			ja:   "フラグに値を指定してください: --space",
			en:   "Specify a value for flag: --space",
		},
		{
			name: "短縮フラグの値の欠落",
			args: []string{"-s"},
			ja:   "フラグに値を指定してください: -s",
			en:   "Specify a value for flag: -s",
		},
		{
			name: "整数の値が不正",
			args: []string{"--limit=abc"},
			ja:   `フラグ--limitの値が不正です: "abc" (型: int)`,
			en:   `Invalid value for flag --limit: "abc" (type: int)`,
		},
		{
			name: "真偽値が不正",
			args: []string{"--help=invalid"},
			ja:   `フラグ--helpの値が不正です: "invalid" (型: bool)`,
			en:   `Invalid value for flag --help: "invalid" (type: bool)`,
		},
		{
			name: "構文が不正",
			args: []string{"--=value"},
			ja:   "フラグの書式が不正です: --=value",
			en:   "Invalid flag syntax: --=value",
		},
	}

	for _, tt := range tests {
		for _, lang := range []string{i18n.LangJa, i18n.LangEn} {
			t.Run(tt.name+"/"+lang, func(t *testing.T) {
				t.Parallel()

				cmd := &cobra.Command{Use: "wikino"}
				cmd.Flags().BoolP("help", "h", false, "")
				cmd.Flags().StringP("space", "s", "", "")
				cmd.Flags().Int("limit", 0, "")
				err := cmd.Flags().Parse(tt.args)
				if err == nil {
					t.Fatal("フラグの解析エラーを期待したが、nilだった")
				}

				loc := i18n.NewLocalizer(lang)
				got := NewFlagError(loc, cmd, fmt.Errorf("フラグの解析: %w", err))
				want := tt.ja
				if lang == i18n.LangEn {
					want = tt.en
				}
				if got.Message != want {
					t.Errorf("メッセージ = %q、期待値 = %q", got.Message, want)
				}
				if wantHint := loc.T("usage_error_hint", map[string]any{"Command": "wikino"}); got.Hint != wantHint {
					t.Errorf("ヒント = %q、期待値 = %q", got.Hint, wantHint)
				}
			})
		}
	}
}

func TestNewFlagErrorUnknownType(t *testing.T) {
	t.Parallel()

	for _, lang := range []string{i18n.LangJa, i18n.LangEn} {
		t.Run(lang, func(t *testing.T) {
			t.Parallel()

			loc := i18n.NewLocalizer(lang)
			got := NewFlagError(loc, &cobra.Command{Use: "wikino"}, errors.New("unclassified error"))
			want := "フラグの指定が誤っています。"
			if lang == i18n.LangEn {
				want = "Invalid flag specification."
			}
			if got.Message != want {
				t.Errorf("メッセージ = %q、期待値 = %q", got.Message, want)
			}
		})
	}
}

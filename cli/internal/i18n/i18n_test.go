package i18n

import (
	"maps"
	"slices"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestDetectLang(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{name: "環境変数が無ければ英語", env: map[string]string{}, want: LangEn},
		{name: "LANGが日本語なら日本語", env: map[string]string{"LANG": "ja_JP.UTF-8"}, want: LangJa},
		{name: "LANGが英語なら英語", env: map[string]string{"LANG": "en_US.UTF-8"}, want: LangEn},
		{name: "日本語・英語以外は英語", env: map[string]string{"LANG": "fr_FR.UTF-8"}, want: LangEn},
		{name: "Cロケールは英語", env: map[string]string{"LANG": "C.UTF-8"}, want: LangEn},
		{name: "言語コードだけのjaも日本語", env: map[string]string{"LANG": "ja"}, want: LangJa},
		{name: "ハイフン区切りのja-JPも日本語", env: map[string]string{"LANG": "ja-JP"}, want: LangJa},
		{name: "大文字のJA_JPも日本語", env: map[string]string{"LANG": "JA_JP.UTF-8"}, want: LangJa},
		{name: "修飾子付きのja_JP@modも日本語", env: map[string]string{"LANG": "ja_JP@mod"}, want: LangJa},
		{name: "jaで始まる別の言語 (jv) は英語", env: map[string]string{"LANG": "jv_ID.UTF-8"}, want: LangEn},
		{name: "jaで始まる3文字の言語コードは英語", env: map[string]string{"LANG": "jam"}, want: LangEn},
		{
			name: "LC_MESSAGESはLANGより優先する",
			env:  map[string]string{"LC_MESSAGES": "ja_JP.UTF-8", "LANG": "en_US.UTF-8"},
			want: LangJa,
		},
		{
			name: "LC_ALLはLC_MESSAGESより優先する",
			env:  map[string]string{"LC_ALL": "en_US.UTF-8", "LC_MESSAGES": "ja_JP.UTF-8"},
			want: LangEn,
		},
		{
			name: "WIKINO_LANGはLC_ALLより優先する",
			env:  map[string]string{"WIKINO_LANG": "ja", "LC_ALL": "en_US.UTF-8"},
			want: LangJa,
		},
		{
			name: "WIKINO_LANGが英語ならロケールが日本語でも英語",
			env:  map[string]string{"WIKINO_LANG": "en", "LANG": "ja_JP.UTF-8"},
			want: LangEn,
		},
		{
			name: "空の環境変数は無視して次を見る",
			env:  map[string]string{"WIKINO_LANG": "", "LC_ALL": "", "LANG": "ja_JP.UTF-8"},
			want: LangJa,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := DetectLang(func(key string) string { return tt.env[key] })
			if got != tt.want {
				t.Errorf("DetectLang() = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}

func TestLocalizerT(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		lang string
		id   string
		data map[string]any
		want string
	}{
		{name: "日本語の文言", lang: LangJa, id: "help_heading_usage", want: "使い方"},
		{name: "英語の文言", lang: LangEn, id: "help_heading_usage", want: "Usage"},
		{
			name: "プレースホルダーを埋める",
			lang: LangJa,
			id:   "usage_error_unknown_subcommand",
			data: map[string]any{"Name": "foo"},
			want: `不明なサブコマンドです: "foo"`,
		},
		{name: "翻訳が無いときはメッセージIDを返す", lang: LangJa, id: "no_such_message", want: "no_such_message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loc := NewLocalizer(tt.lang)

			got := loc.T(tt.id, tt.data)
			if got != tt.want {
				t.Errorf("T(%q) = %q、期待値 = %q", tt.id, got, tt.want)
			}
		})
	}
}

// TestLocalesHaveSameKeysは、日本語と英語の翻訳ファイルが同じキーを持つことを確かめる。
// 英語の翻訳が欠けると、go-i18nは既定の日本語で黙って補うため、表示を見ても気づきにくい。
func TestLocalesHaveSameKeys(t *testing.T) {
	t.Parallel()

	ja := readLocaleKeys(t, LangJa)
	en := readLocaleKeys(t, LangEn)

	if !slices.Equal(ja, en) {
		t.Errorf("翻訳ファイルのキーが一致しない。ja = %v、en = %v", ja, en)
	}
}

func readLocaleKeys(t *testing.T, lang string) []string {
	t.Helper()

	data, err := localesFS.ReadFile("locales/" + lang + ".toml")
	if err != nil {
		t.Fatalf("翻訳ファイルを読み込めない: %v", err)
	}

	var messages map[string]any
	if err := toml.Unmarshal(data, &messages); err != nil {
		t.Fatalf("翻訳ファイルを解析できない: %v", err)
	}

	return slices.Sorted(maps.Keys(messages))
}

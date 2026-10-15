// Package i18nはCLIが利用者に出す文言 (ヘルプ・エラー・確認) の翻訳を提供する。
//
// サーバー (go/internal/i18n) と同じくgo-i18nとTOMLで管理し、日本語を正本とする。
// サーバーはリクエストごとのcontextでロケールを運ぶが、CLIは1回の実行でロケールが
// 変わらないため、実行の最初に判定したLocalizerをコマンドの組み立てに渡す。
package i18n

import (
	"embed"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed locales/*.toml
var localesFS embed.FS

// 対応する言語
const (
	LangJa = "ja"
	LangEn = "en"
)

// langEnvKeysはロケールの判定に使う環境変数で、先に並べたものを優先する。
// WIKINO_LANGでCLIだけの言語を指定でき、それ以降はPOSIXのロケールの優先順位に従う。
var langEnvKeys = []string{"WIKINO_LANG", "LC_ALL", "LC_MESSAGES", "LANG"}

var bundle *i18n.Bundle

func init() {
	bundle = i18n.NewBundle(language.Japanese)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	for _, lang := range []string{LangJa, LangEn} {
		name := fmt.Sprintf("%s.toml", lang)

		data, err := localesFS.ReadFile("locales/" + name)
		if err != nil {
			panic(fmt.Sprintf("翻訳ファイル%sを読み込めません: %v", name, err))
		}

		bundle.MustParseMessageFileBytes(data, name)
	}
}

// DetectLangは環境変数から表示言語を判定する。
// 値が設定されている最初の環境変数で決め、日本語以外はすべて英語にする。
// getenvを受け取るのは、テストでプロセスの環境変数を書き換えずに判定を確かめるため。
func DetectLang(getenv func(string) string) string {
	for _, key := range langEnvKeys {
		value := getenv(key)
		if value == "" {
			continue
		}

		if isJapanese(value) {
			return LangJa
		}

		return LangEn
	}

	return LangEn
}

// isJapaneseはロケールの値 (ja・ja_JP.UTF-8・ja-JPなど) が日本語を指すかを返す。
// 言語コードの区切りまでを比べ、jaで始まる別の言語 (jv・javなど) を日本語と取り違えない。
func isJapanese(value string) bool {
	code, _, _ := strings.Cut(strings.ToLower(value), ".")
	code, _, _ = strings.Cut(code, "@")
	code, _, _ = strings.Cut(code, "_")
	code, _, _ = strings.Cut(code, "-")

	return code == LangJa
}

// Localizerは1つの言語で文言を翻訳する。
type Localizer struct {
	lang      string
	localizer *i18n.Localizer
}

// NewLocalizerはlangで翻訳するLocalizerを返す。
func NewLocalizer(lang string) *Localizer {
	return &Localizer{
		lang:      lang,
		localizer: i18n.NewLocalizer(bundle, lang),
	}
}

// Langは翻訳に使う言語を返す。
func (l *Localizer) Lang() string {
	return l.lang
}

// TはmessageIDの文言を翻訳する。templateDataで文言のプレースホルダーを埋める。
// 翻訳が見つからないときはmessageIDをそのまま返し、文言の欠落をテストと画面で気づけるようにする。
func (l *Localizer) T(messageID string, templateData ...map[string]any) string {
	config := &i18n.LocalizeConfig{MessageID: messageID}
	if len(templateData) > 0 && templateData[0] != nil {
		config.TemplateData = templateData[0]
	}

	message, err := l.localizer.Localize(config)
	if err != nil {
		return messageID
	}

	return message
}

package exportfile

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"go.yaml.in/yaml/v3"
)

// titleKeyは、エクスポートが変換前のタイトルを書き出すfrontmatterのキー。Wikinoが書く
// キーはすべて "wikino_" を接頭辞として持つ。frontmatterは書き手の本文と同じファイルに同居し、
// アーカイブを開く側のツールもそこへ独自のキーを置くためである。接頭辞があれば、どのキーを
// Wikinoが書いたのかが分かり、読む側が別の意味で使っているキーとも衝突しない。
const titleKey = "wikino_title"

// frontmatterDelimiterはYAML frontmatterのブロックを開き、また閉じる区切り。
const frontmatterDelimiter = "---"

// WithFrontmatterは、titleをYAML frontmatterに持たせたbodyを返す。アーカイブはこれを
// ページのMarkdownファイルとして書き出す。そのファイルの名前は置換・重複の解決・切り詰めを
// 経ており、もうタイトルを綴っていない。タイトルが完全な形で残る場所がfrontmatterである。
//
// 有効なmappingにはキーを追加し、既存タイトルは他のプロパティを保持して置き換える。
// 不正なYAMLや水平線の間の通常の文章は、新しいブロックの下に本文として残す。
func WithFrontmatter(body, title string) string {
	opening, closing, document, ok := frontmatterBlock(body)
	if ok {
		mapping := document.Content[0]
		found := false
		for i := 0; i < len(mapping.Content); i += 2 {
			if mapping.Content[i].Value == titleKey {
				value := mapping.Content[i+1]
				value.Kind, value.Tag, value.Value = yaml.ScalarNode, "!!str", title
				value.Style = yaml.DoubleQuotedStyle
				value.Content, value.Alias = nil, nil
				found = true
				break
			}
		}
		if !found {
			mapping.Content = append([]*yaml.Node{
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: titleKey},
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: title, Style: yaml.DoubleQuotedStyle},
			}, mapping.Content...)
		}
		var encoded bytes.Buffer
		encoder := yaml.NewEncoder(&encoded)
		encoder.SetIndent(2)
		if err := encoder.Encode(document); err == nil {
			if err := encoder.Close(); err == nil {
				return body[:opening] + matchLineEndings(encoded.String(), body[:opening]) + body[closing:]
			}
		}
	}

	block := frontmatterDelimiter + "\n" + titleKey + ": " + quoteYAML(title) + "\n" + frontmatterDelimiter + "\n\n"

	return matchLineEndings(block, body) + body
}

// matchLineEndingsは、referenceが既に使っている改行コードでblockを書き直して返す。
// 使っている改行コードはreferenceの最初の改行から判断する。
//
// YAMLのエンコーダーもここで組み立てるブロックもLFしか書かないが、ブラウザのフォームから
// 送られた本文はCRLFを持つ。これが無いと、エクスポートが書き出すファイルの改行コードが
// frontmatterの中と外で食い違う。
func matchLineEndings(block, reference string) string {
	i := strings.IndexByte(reference, '\n')
	if i <= 0 || reference[i-1] != '\r' {
		return block
	}

	return strings.ReplaceAll(block, "\n", "\r\n")
}

// frontmatterBlockは先頭の区切り内にある単一の有効なYAML mappingを判定する。
// mappingでない値や不正なYAMLは本文に残す。デコードでキーの重複も検証する。
func frontmatterBlock(body string) (int, int, *yaml.Node, bool) {
	rest, ok := strings.CutPrefix(body, frontmatterDelimiter+"\n")
	if !ok {
		rest, ok = strings.CutPrefix(body, frontmatterDelimiter+"\r\n")
	}
	if !ok {
		return 0, 0, nil, false
	}
	opening := len(body) - len(rest)
	closing := opening
	for _, line := range strings.Split(rest, "\n") {
		if strings.TrimSuffix(line, "\r") == frontmatterDelimiter {
			var document yaml.Node
			decoder := yaml.NewDecoder(strings.NewReader(body[opening:closing]))
			if err := decoder.Decode(&document); err != nil || len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
				return 0, 0, nil, false
			}
			var values map[string]any
			if err := document.Decode(&values); err != nil {
				return 0, 0, nil, false
			}
			var extra yaml.Node
			if err := decoder.Decode(&extra); err != io.EOF {
				return 0, 0, nil, false
			}
			return opening, closing, &document, true
		}
		closing += len(line) + 1
	}
	return 0, 0, nil, false
}

// quoteYAMLはsを二重引用符のYAMLスカラーとして返す。YAMLはJSONの上位集合なので
// JSONの文字列リテラルはそのままYAMLのスカラーであり、どの文字をエスケープすべきかの判断は
// encoding/jsonに委ねられる。HTMLのエスケープを切っているのは、有効にすると "<" ">" "&" が
// \u003cのような形で残り、人が読むためのタイトルとしては雑音になるためである。
func quoteYAML(s string) string {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(s); err != nil {
		// 文字列のエンコードは失敗しないが、空のスカラーもfrontmatterの値としては有効である。
		return `""`
	}

	return strings.TrimSuffix(buf.String(), "\n")
}

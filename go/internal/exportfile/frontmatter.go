package exportfile

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"go.yaml.in/yaml/v3"
)

// titleKey is the frontmatter key an export writes the untranslated title under. Every key Wikino
// writes carries the "wikino_" prefix: the frontmatter shares a file with what the author wrote,
// and the tool the archive is opened in puts its own keys there too. The prefix says which keys
// came from Wikino, and keeps them clear of a key the reading side already uses for its own
// meaning.
//
// [Ja] titleKey は、エクスポートが変換前のタイトルを書き出す frontmatter のキー。Wikino が書く
// キーはすべて "wikino_" を接頭辞として持つ。frontmatter は書き手の本文と同じファイルに同居し、
// アーカイブを開く側のツールもそこへ独自のキーを置くためである。接頭辞があれば、どのキーを
// Wikino が書いたのかが分かり、読む側が別の意味で使っているキーとも衝突しない。
const titleKey = "wikino_title"

// frontmatterDelimiter opens and closes a YAML frontmatter block.
//
// [Ja] frontmatterDelimiter は YAML frontmatter のブロックを開き、また閉じる区切り。
const frontmatterDelimiter = "---"

// WithFrontmatter returns body carrying title in its YAML frontmatter, which is what the archive
// writes as the Markdown file of a page. The name of that file has been through replacement,
// deduplication and truncation, so it no longer spells the title out; the frontmatter is where
// the title survives in full.
//
// A valid mapping receives the key, replacing an existing title while preserving other properties.
// Invalid YAML and ordinary text between thematic breaks remain body text under a new block.
//
// [Ja] WithFrontmatter は、title を YAML frontmatter に持たせた body を返す。アーカイブはこれを
// ページの Markdown ファイルとして書き出す。そのファイルの名前は置換・重複の解決・切り詰めを
// 経ており、もうタイトルを綴っていない。タイトルが完全な形で残る場所が frontmatter である。
//
// 有効な mapping にはキーを追加し、既存タイトルは他のプロパティを保持して置き換える。
// 不正な YAML や水平線の間の通常の文章は、新しいブロックの下に本文として残す。
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

// matchLineEndings returns block written with the line ending reference already uses, which is
// taken from its first line break.
//
// Both the YAML encoder and the block built here write LF only, while a body submitted through a
// browser form carries CRLF. Without this, the file an export writes would use one line ending
// inside its frontmatter and another around it.
//
// [Ja] matchLineEndings は、reference が既に使っている改行コードで block を書き直して返す。
// 使っている改行コードは reference の最初の改行から判断する。
//
// YAML のエンコーダーもここで組み立てるブロックも LF しか書かないが、ブラウザのフォームから
// 送られた本文は CRLF を持つ。これが無いと、エクスポートが書き出すファイルの改行コードが
// frontmatter の中と外で食い違う。
func matchLineEndings(block, reference string) string {
	i := strings.IndexByte(reference, '\n')
	if i <= 0 || reference[i-1] != '\r' {
		return block
	}

	return strings.ReplaceAll(block, "\n", "\r\n")
}

// frontmatterBlock recognizes a single valid YAML mapping between leading delimiters.
// Non-mappings and invalid YAML remain body text; decoding also checks duplicate keys.
//
// [Ja] frontmatterBlock は先頭の区切り内にある単一の有効な YAML mapping を判定する。
// mapping でない値や不正な YAML は本文に残す。デコードでキーの重複も検証する。
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

// quoteYAML returns s as a double-quoted YAML scalar. YAML is a superset of JSON, so a JSON string
// literal is one already, and encoding/json is what decides which characters have to be escaped.
// HTML escaping is turned off because it would leave "<", ">" and "&" as \u003c and the like,
// which reads as noise in a title a person is meant to be able to read.
//
// [Ja] quoteYAML は s を二重引用符の YAML スカラーとして返す。YAML は JSON の上位集合なので
// JSON の文字列リテラルはそのまま YAML のスカラーであり、どの文字をエスケープすべきかの判断は
// encoding/json に委ねられる。HTML のエスケープを切っているのは、有効にすると "<" ">" "&" が
// \u003c のような形で残り、人が読むためのタイトルとしては雑音になるためである。
func quoteYAML(s string) string {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(s); err != nil {
		// Encoding a string cannot fail, but an empty scalar is still a valid frontmatter value.
		//
		// [Ja] 文字列のエンコードは失敗しないが、空のスカラーも frontmatter の値としては有効である。
		return `""`
	}

	return strings.TrimSuffix(buf.String(), "\n")
}

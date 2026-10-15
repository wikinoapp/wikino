package pagefile

import (
	"errors"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// 書き戻しで読むWikinoのキー。これ以外の `wikino_` のキー (公開状態・更新日時・URLなど) は
// サーバーが決める読み取り専用の値のため、読まずに取り除く。
const (
	keyTitle  = keyPrefix + "title"
	keyNumber = keyPrefix + "number"
	keyETag   = keyPrefix + "etag"
)

// ErrUnremovableKeysは、本文自身のfrontmatterから `wikino_` のキーを取り除くと、
// 他のプロパティが変わってしまうことを表す (エイリアスで値を共有している場合など)。
// 書き出し (WithFrontmatter) はこのようなブロックに統合せず新しいブロックを置くため、手で書いたファイルでだけ起きる。
var ErrUnremovableKeys = errors.New("pagefile: wikino_のキーを取り除けないfrontmatterです")

// InvalidValueErrorは、Wikinoのキーの値の形が違うことを表す。
type InvalidValueError struct {
	// Keyはキーの名前 (例: wikino_number)。
	Key string
}

func (e *InvalidValueError) Error() string {
	return fmt.Sprintf("pagefile: %sの値の形が違います", e.Key)
}

// Fileは、読み込んだページファイル。
type File struct {
	// Titleは `wikino_title` の値。キーが無ければnil。
	Title *string
	// Numberは `wikino_number` の値。キーが無ければnil。
	Number *int32
	// ETagは `wikino_etag` の値。キーが無ければ空文字列。
	ETag string
	// BodyはWikinoのキーを取り除いた本文。
	Body string
}

// Parseは、Wikinoファイル形式のページファイルを読む。WithFrontmatterの逆の操作で、
// WithFrontmatterで書き出したファイルを読むと、書き出す前の本文が戻る
// (本文自身のfrontmatterにあった `wikino_` のキーは、書き出しで取り除かれるため戻らない)。
//
// Wikinoのキーは先頭のfrontmatterのブロックからだけ読む。ブロックが `wikino_` で始まるキーだけを
// 持つなら、書き出しが本文の前に置いたブロックとして、ブロックと直後の空行を取り除く。
// `wikino_` で始まらないキーを持つなら、本文自身のfrontmatterとして、`wikino_` のキーの行だけを取り除き、
// ブロックの他の行は1バイトも変えない。先頭に有効なfrontmatterが無ければ、全体を本文とする。
func Parse(content string) (File, error) {
	opening, closing, document, ok := frontmatterBlock(content)
	if !ok {
		return File{Body: content}, nil
	}
	mapping := document.Content[0]

	var file File
	ownKeys := false
	hasWikinoKeys := false
	for i := 0; i < len(mapping.Content); i += 2 {
		key, value := mapping.Content[i], mapping.Content[i+1]
		if !strings.HasPrefix(key.Value, keyPrefix) {
			ownKeys = true
			continue
		}
		hasWikinoKeys = true

		var err error
		switch key.Value {
		case keyTitle:
			var title string
			title, err = stringValue(key.Value, value)
			file.Title = &title
		case keyNumber:
			var number int32
			number, err = numberValue(key.Value, value)
			file.Number = &number
		case keyETag:
			file.ETag, err = stringValue(key.Value, value)
		}
		if err != nil {
			return File{}, err
		}
	}
	if !hasWikinoKeys {
		return File{Body: content}, nil
	}

	if !ownKeys {
		file.Body = content[afterLine(content, closing):]
		file.Body = trimLeadingNewline(file.Body)
		return file, nil
	}

	block := content[opening:closing]
	lines := blockLines(block)
	removed, ok := wikinoEntryLines(lines, mapping)
	if !ok {
		return File{}, ErrUnremovableKeys
	}
	var stripped strings.Builder
	for i, line := range lines {
		if !removed[i] {
			stripped.WriteString(line)
		}
	}
	if !sameProperties(block, stripped.String(), nil) {
		return File{}, ErrUnremovableKeys
	}
	file.Body = content[:opening] + stripped.String() + content[closing:]

	return file, nil
}

// stringValueは、文字列のスカラーの値を返す。
func stringValue(key string, value *yaml.Node) (string, error) {
	if value.Kind != yaml.ScalarNode || value.Tag != "!!str" {
		return "", &InvalidValueError{Key: key}
	}

	return value.Value, nil
}

// numberValueは、32ビットに収まる整数のスカラーの値を返す。
func numberValue(key string, value *yaml.Node) (int32, error) {
	var number int32
	if value.Kind != yaml.ScalarNode || value.Tag != "!!int" || value.Decode(&number) != nil {
		return 0, &InvalidValueError{Key: key}
	}

	return number, nil
}

// afterLineは、offsetから始まる行の次の行の先頭の位置を返す。最後の行なら末尾の位置を返す。
func afterLine(s string, offset int) int {
	i := strings.IndexByte(s[offset:], '\n')
	if i < 0 {
		return len(s)
	}

	return offset + i + 1
}

// trimLeadingNewlineは、先頭の1つの改行 (LFかCRLF) を取り除く。
// 書き出しは、本文の前に置いたブロックと本文の間に空行を1行だけ挟むため、それだけを取り除く。
func trimLeadingNewline(s string) string {
	if rest, ok := strings.CutPrefix(s, "\r\n"); ok {
		return rest
	}

	return strings.TrimPrefix(s, "\n")
}

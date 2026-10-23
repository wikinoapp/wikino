package exportfile

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// keyPrefixは、Wikinoがfrontmatterに書くキーの接頭辞 (Wikinoファイル形式。ADR 0025)。
// frontmatterは書き手の本文と同じファイルに同居し、ファイルを開く側のツールもそこへ独自の
// キーを置くためである。接頭辞があれば、どのキーをWikinoが書いたのかが分かり、読む側が別の
// 意味で使っているキーとも衝突しない。
const keyPrefix = "wikino_"

// frontmatterDelimiterはYAML frontmatterのブロックを開き、また閉じる区切り。
const frontmatterDelimiter = "---"

// PageMetadataは、ページファイルのfrontmatterにWikinoが書く値。値の形は公開APIのページの
// 応答に揃える。CLIの `--md` も同じページファイルを書き出し、両者を共通のテストデータ
// (リポジトリ直下の `testdata/wikino-file-format/`) で検査するためである。
type PageMetadata struct {
	// Titleはページのタイトル。nilならキーを書かない (読む側はタイトルを変えない)
	Title       *string
	Number      int32
	ETag        string
	TopicNumber int32
	Status      string
	ModifiedAt  time.Time
	URL         string
}

// linesはWikinoのキーの行を、書き出す順に改行無しで返す。値の文字列はすべて二重引用符で
// 引用する。キーごとに引用の要否を判断すると、同じ規則をCLIでも実装し直すことになるためである。
func (m PageMetadata) lines() []string {
	var lines []string
	if m.Title != nil {
		lines = append(lines, keyPrefix+"title: "+quoteYAML(*m.Title))
	}
	return append(lines,
		keyPrefix+"number: "+strconv.FormatInt(int64(m.Number), 10),
		keyPrefix+"etag: "+quoteYAML(m.ETag),
		keyPrefix+"topic_number: "+strconv.FormatInt(int64(m.TopicNumber), 10),
		keyPrefix+"status: "+quoteYAML(m.Status),
		keyPrefix+"modified_at: "+quoteYAML(m.ModifiedAt.UTC().Format(time.RFC3339Nano)),
		keyPrefix+"url: "+quoteYAML(m.URL),
	)
}

// WithFrontmatterは、metadataをYAML frontmatterに持たせたbodyを返す。これがWikinoファイル
// 形式のページファイルになる。
//
// bodyが有効なYAMLのmappingのfrontmatterで始まる場合は、そのブロックにWikinoのキーの行を
// 足し、既にある `wikino_` のキーの行を取り除く。YAMLを再エンコードしないのは、本文自身の
// frontmatterの引用符・インデント・コメントを1バイトも変えないためである (変わると、CLIの
// 書き戻しで利用者が触れていない本文に版と差分が生まれる)。
// 行の足し引きで同じmappingを保てないブロック (フロー形式・エイリアスの参照先を消す場合など) と、
// 不正なYAMLや水平線の間の通常の文章には、新しいブロックを本文の前に置く。
func WithFrontmatter(body string, metadata PageMetadata) string {
	lines := metadata.lines()
	if merged, ok := mergeFrontmatter(body, lines); ok {
		return merged
	}

	block := frontmatterDelimiter + "\n" + strings.Join(lines, "\n") + "\n" + frontmatterDelimiter + "\n\n"

	return matchLineEndings(block, body) + body
}

// mergeFrontmatterは、本文自身のfrontmatterのブロックにWikinoのキーの行を足した本文を返す。
// キーの行は先頭のキーの行の直前に置き、タグ・アンカー・コメントの行はその前に残す。
//
// 行の操作はキーの位置だけを見て行うため、結果のブロックを読み直し、Wikinoのキー以外の
// プロパティが元と同じで、Wikinoのキーが意図した値であることを確かめる。確かめられなければ
// falseを返し、呼び出し側は新しいブロックを置く。
func mergeFrontmatter(body string, keyLines []string) (string, bool) {
	opening, closing, document, ok := frontmatterBlock(body)
	if !ok {
		return "", false
	}
	mapping := document.Content[0]
	if mapping.Style&yaml.FlowStyle != 0 {
		return "", false
	}

	block := body[opening:closing]
	lines := strings.SplitAfter(block, "\n")
	lines = lines[:len(lines)-1] // ブロックは区切りの行の直前、つまり改行で終わる

	// 次のキーまでの範囲から、値に含まれない末尾のコメントと空行を除いて削除する
	removed := make([]bool, len(lines))
	for i := 0; i < len(mapping.Content); i += 2 {
		start := mapping.Content[i].Line - 1
		end := len(lines)
		if i+2 < len(mapping.Content) {
			end = mapping.Content[i+2].Line - 1
		}
		if start < 0 || end <= start || end > len(lines) {
			return "", false
		}
		if strings.HasPrefix(mapping.Content[i].Value, keyPrefix) {
			end, ok = frontmatterEntryEnd(lines, start, end, mapping.Content[i], mapping.Content[i+1])
			if !ok {
				return "", false
			}
			for j := start; j < end; j++ {
				removed[j] = true
			}
		}
	}

	newline := matchLineEndings("\n", body)
	insertAt := mapping.Content[0].Line - 1
	var merged strings.Builder
	for i, line := range lines {
		if i == insertAt {
			for _, keyLine := range keyLines {
				merged.WriteString(keyLine + newline)
			}
		}
		if !removed[i] {
			merged.WriteString(line)
		}
	}

	if !sameProperties(block, merged.String(), keyLines) {
		return "", false
	}

	return body[:opening] + merged.String() + body[closing:], true
}

// frontmatterEntryEndはキーと値を削除する範囲の終端を返す。
// 末尾のコメントと空行をいったん範囲から外し、元の値を保つ最短の範囲を読み直して求める。
// インデントだけでは、複数行スカラーのコメント風の行や `|+` の末尾の空行を区別できないためである。
// エイリアスなど、エントリ単独では値を確認できない場合は統合せず元のブロックを残す。
func frontmatterEntryEnd(lines []string, start, end int, key, value *yaml.Node) (int, bool) {
	candidate := end
	for candidate > start+1 {
		line := strings.TrimSpace(lines[candidate-1])
		if line != "" && !strings.HasPrefix(line, "#") {
			break
		}
		candidate--
	}
	if candidate == end {
		return end, true
	}

	var want any
	if err := value.Decode(&want); err != nil {
		return 0, false
	}
	for ; candidate <= end; candidate++ {
		var entry map[string]any
		if err := yaml.Unmarshal([]byte(strings.Join(lines[start:candidate], "")), &entry); err != nil {
			continue
		}
		got, ok := entry[key.Value]
		if ok && len(entry) == 1 && reflect.DeepEqual(got, want) {
			return candidate, true
		}
	}
	return 0, false
}

// samePropertiesは、mergedがoriginalのWikinoのキー以外のプロパティと、keyLinesの
// プロパティをちょうど持つかを返す。
func sameProperties(original, merged string, keyLines []string) bool {
	var want map[string]any
	if err := yaml.Unmarshal([]byte(original), &want); err != nil {
		return false
	}
	for key := range want {
		if strings.HasPrefix(key, keyPrefix) {
			delete(want, key)
		}
	}
	var keys map[string]any
	if err := yaml.Unmarshal([]byte(strings.Join(keyLines, "\n")), &keys); err != nil {
		return false
	}
	for key, value := range keys {
		want[key] = value
	}

	var got map[string]any
	if err := yaml.Unmarshal([]byte(merged), &got); err != nil {
		return false
	}

	return reflect.DeepEqual(got, want)
}

// matchLineEndingsは、referenceが既に使っている改行コードでblockを書き直して返す。
// 使っている改行コードはreferenceの最初の改行から判断する。
//
// ここで組み立てるブロックとキーの行はLFしか書かないが、ブラウザのフォームから送られた本文は
// CRLFを持つ。これが無いと、書き出すファイルの改行コードがfrontmatterの中と外で食い違う。
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
// \u003cのような形で残り、人が読むための値としては雑音になるためである。
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

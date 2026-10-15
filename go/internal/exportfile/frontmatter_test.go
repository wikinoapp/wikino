package exportfile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode"

	"go.yaml.in/yaml/v3"
)

// pageFileCaseは、リポジトリ直下の共通のテストデータ (`testdata/wikino-file-format/page-files.json`)
// の1件。CLIの `--md` も同じテストデータで検査する
type pageFileCase struct {
	Name string `json:"name"`
	Page struct {
		Title       *string `json:"title"`
		Number      int32   `json:"number"`
		ETag        string  `json:"etag"`
		TopicNumber int32   `json:"topic_number"`
		Status      string  `json:"status"`
		ModifiedAt  string  `json:"modified_at"`
		URL         string  `json:"url"`
		Body        string  `json:"body"`
	} `json:"page"`
	Want string `json:"want"`
}

// readPageFilesは共通のテストデータの生のバイト列を読む
func readPageFiles(t *testing.T) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "wikino-file-format", "page-files.json"))
	if err != nil {
		t.Fatalf("共通のテストデータを読めません: %v", err)
	}
	return data
}

// loadPageFileCasesは共通のテストデータを読む
func loadPageFileCases(t *testing.T) []pageFileCase {
	t.Helper()

	data := readPageFiles(t)
	var cases []pageFileCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("共通のテストデータを解釈できません: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("共通のテストデータにケースがありません")
	}
	return cases
}

// metadataはケースのページから、frontmatterに書く値を作る
func (c pageFileCase) metadata(t *testing.T) PageMetadata {
	t.Helper()

	modifiedAt, err := time.Parse(time.RFC3339Nano, c.Page.ModifiedAt)
	if err != nil {
		t.Fatalf("modified_atを解釈できません: %v", err)
	}
	return PageMetadata{
		Title:       c.Page.Title,
		Number:      c.Page.Number,
		ETag:        c.Page.ETag,
		TopicNumber: c.Page.TopicNumber,
		Status:      c.Page.Status,
		ModifiedAt:  modifiedAt,
		URL:         c.Page.URL,
	}
}

func TestWithFrontmatter_SharedTestdata(t *testing.T) {
	t.Parallel()

	for _, tt := range loadPageFileCases(t) {
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			if got := WithFrontmatter(tt.Page.Body, tt.metadata(t)); got != tt.Want {
				t.Errorf("WithFrontmatter() = %q、期待値 = %q", got, tt.Want)
			}
		})
	}
}

// テストデータの期待値そのものが、先頭のブロックにページの値を持つ有効なfrontmatterであることを確かめる。
// 期待値を手で足したときの誤りを、CLIの側で初めて見つけることにならないようにする
func TestWithFrontmatter_SharedTestdataIsValid(t *testing.T) {
	t.Parallel()

	for _, tt := range loadPageFileCases(t) {
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			opening, closing, _, ok := frontmatterBlock(tt.Want)
			if !ok {
				t.Fatalf("期待値の先頭が有効なfrontmatterではありません: %q", tt.Want)
			}
			var properties map[string]any
			if err := yaml.Unmarshal([]byte(tt.Want[opening:closing]), &properties); err != nil {
				t.Fatalf("frontmatterを読めません: %v", err)
			}

			metadata := tt.metadata(t)
			want := map[string]any{
				"wikino_number":       int(metadata.Number),
				"wikino_etag":         metadata.ETag,
				"wikino_topic_number": int(metadata.TopicNumber),
				"wikino_status":       metadata.Status,
				"wikino_modified_at":  metadata.ModifiedAt.UTC().Format(time.RFC3339Nano),
				"wikino_url":          metadata.URL,
			}
			if metadata.Title != nil {
				want["wikino_title"] = *metadata.Title
			} else if _, ok := properties["wikino_title"]; ok {
				t.Errorf("タイトルの無いページにwikino_titleがあります: %v", properties)
			}
			for key, value := range want {
				if properties[key] != value {
					t.Errorf("%s = %#v、期待値 = %#v", key, properties[key], value)
				}
			}
		})
	}
}

// テストデータのBOMや制御文字は、JSONのエスケープで書かれていることを確かめる。
// 文字そのものはdiffやエディタで見えず、編集の過程で消えたり紛れ込んだりしても気付けないため
func TestWithFrontmatter_SharedTestdataEscapesInvisibleCharacters(t *testing.T) {
	t.Parallel()

	for i, r := range string(readPageFiles(t)) {
		if r != '\n' && (unicode.IsControl(r) || unicode.Is(unicode.Cf, r)) {
			t.Errorf("%dバイト目に見えない文字 %U がそのまま書かれています。JSONのエスケープで書いてください", i, r)
		}
	}
}

func TestWithFrontmatter_ReplacesTitleKeyOnce(t *testing.T) {
	t.Parallel()

	title := "新しい"
	got := WithFrontmatter("---\nwikino_title: 古い\nwikino_title_note: 残らない\naliases: [other]\n---\n\n本文\n", PageMetadata{
		Title:      &title,
		Number:     1,
		ETag:       `"abc"`,
		Status:     "published",
		ModifiedAt: time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC),
		URL:        "https://wikino.example.com/s/example/pages/1",
	})

	want := "---\n" +
		"wikino_title: \"新しい\"\n" +
		"wikino_number: 1\n" +
		"wikino_etag: \"\\\"abc\\\"\"\n" +
		"wikino_topic_number: 0\n" +
		"wikino_status: \"published\"\n" +
		"wikino_modified_at: \"2026-10-01T09:00:00Z\"\n" +
		"wikino_url: \"https://wikino.example.com/s/example/pages/1\"\n" +
		"aliases: [other]\n" +
		"---\n\n本文\n"
	if got != want {
		t.Errorf("WithFrontmatter() = %q、期待値 = %q", got, want)
	}
}

func TestWithFrontmatter_PreservesNonMappings(t *testing.T) {
	t.Parallel()

	title := "current"
	metadata := PageMetadata{Title: &title, Status: "published"}
	block := "---\n" + strings.Join(metadata.lines(), "\n") + "\n---\n\n"

	for _, body := range []string{
		"---\n\nordinary paragraph\n\n---\n\nbody",
		"---\n- list item\n---\nbody",
		"---\ninvalid: [\n---\nbody",
		"---\na: 1\na: 2\n---\nbody",
		"---\n{}\n...\nstray content\n---\nbody",
		"---\n---\nbody",
	} {
		t.Run(body, func(t *testing.T) {
			t.Parallel()

			if got := WithFrontmatter(body, metadata); got != block+body {
				t.Errorf("本文がメタデータとして消費された: %q", got)
			}
		})
	}
}

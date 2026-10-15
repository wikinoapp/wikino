package pagefile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// pageFileCaseは、リポジトリ直下の共通のテストデータ (`testdata/wikino-file-format/page-files.json`)
// の1件。サーバーのエクスポートも同じテストデータで検査し、両者が同じページファイルを書き出すことを確かめる。
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

func loadPageFileCases(t *testing.T) []pageFileCase {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "wikino-file-format", "page-files.json"))
	if err != nil {
		t.Fatalf("共通のテストデータを読めません: %v", err)
	}
	var cases []pageFileCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("共通のテストデータを解釈できません: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("共通のテストデータにケースがありません")
	}

	return cases
}

func TestWithFrontmatter_SharedTestdata(t *testing.T) {
	t.Parallel()

	for _, tt := range loadPageFileCases(t) {
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			modifiedAt, err := time.Parse(time.RFC3339Nano, tt.Page.ModifiedAt)
			if err != nil {
				t.Fatalf("modified_atを解釈できません: %v", err)
			}
			metadata := Metadata{
				Title:       tt.Page.Title,
				Number:      &tt.Page.Number,
				ETag:        tt.Page.ETag,
				TopicNumber: &tt.Page.TopicNumber,
				Status:      tt.Page.Status,
				ModifiedAt:  &modifiedAt,
				URL:         tt.Page.URL,
			}

			if got := WithFrontmatter(tt.Page.Body, metadata); got != tt.Want {
				t.Errorf("WithFrontmatter() = %q、期待値 = %q", got, tt.Want)
			}
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}

// 下書きとリビジョンは応答にある項目だけをキーにする。共通のテストデータはページだけを扱うため、ここで確かめる。
func TestWithFrontmatter_DraftAndRevision(t *testing.T) {
	t.Parallel()

	modifiedAt := time.Date(2026, 10, 1, 9, 0, 0, 123000000, time.UTC)

	tests := []struct {
		name     string
		body     string
		metadata Metadata
		want     string
	}{
		{
			name: "下書きは公開状態を書かない",
			body: "本文\n",
			metadata: Metadata{
				Title:       ptr("議事録"),
				Number:      ptr(int32(12)),
				ETag:        `"draft-etag"`,
				TopicNumber: ptr(int32(3)),
				ModifiedAt:  &modifiedAt,
				URL:         "https://wikino.example.com/s/example/pages/12/edit",
			},
			want: "---\n" +
				"wikino_title: \"議事録\"\n" +
				"wikino_number: 12\n" +
				"wikino_etag: \"\\\"draft-etag\\\"\"\n" +
				"wikino_topic_number: 3\n" +
				"wikino_modified_at: \"2026-10-01T09:00:00.123Z\"\n" +
				"wikino_url: \"https://wikino.example.com/s/example/pages/12/edit\"\n" +
				"---\n\n本文\n",
		},
		{
			name:     "リビジョンはタイトルだけを書き、ETagを書かない",
			body:     "本文\n",
			metadata: Metadata{Title: ptr("議事録")},
			want:     "---\nwikino_title: \"議事録\"\n---\n\n本文\n",
		},
		{
			name: "リビジョンのタイトルを本文自身のfrontmatterに足し、既存のwikino_のキーを取り除く",
			body: "---\ntags: [a, b]\nwikino_etag: \"old\"\n---\n本文\n",
			metadata: Metadata{
				Title: ptr("議事録"),
			},
			want: "---\nwikino_title: \"議事録\"\ntags: [a, b]\n---\n本文\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := WithFrontmatter(tt.body, tt.metadata); got != tt.want {
				t.Errorf("WithFrontmatter() = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}

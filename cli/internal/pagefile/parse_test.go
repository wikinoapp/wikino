package pagefile

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// 共通のテストデータの書き出しの期待値を読み、書き出す前の本文とWikinoのキーが戻ることを確かめる。
// `--md` で書き出したファイルを `--md-file` で書き戻しても、本文が変わらないことの検査になる。
func TestParse_SharedTestdata(t *testing.T) {
	t.Parallel()

	for _, tt := range loadPageFileCases(t) {
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			got, err := Parse(tt.Want)
			if err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}

			if got.ETag != tt.Page.ETag {
				t.Errorf("ETag = %q、期待値 = %q", got.ETag, tt.Page.ETag)
			}
			if got.Number == nil || *got.Number != tt.Page.Number {
				t.Errorf("Number = %v、期待値 = %d", got.Number, tt.Page.Number)
			}
			switch {
			case tt.Page.Title == nil && got.Title != nil:
				t.Errorf("Title = %q、期待値 = nil", *got.Title)
			case tt.Page.Title != nil && (got.Title == nil || *got.Title != *tt.Page.Title):
				t.Errorf("Title = %v、期待値 = %q", got.Title, *tt.Page.Title)
			}

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

			// 本文自身のfrontmatterにあった `wikino_` のキーは書き出しで取り除かれ、元の本文には戻らない。
			// それ以外の本文は、書き出す前の本文に戻る。
			if !strings.Contains(tt.Page.Body, keyPrefix) && got.Body != tt.Page.Body {
				t.Errorf("Body = %q、期待値 = %q", got.Body, tt.Page.Body)
			}

			// 読んだ本文を書き出して読み直しても、本文は変わらない (書き戻しを繰り返しても本文がずれない)。
			again, err := Parse(WithFrontmatter(got.Body, metadata))
			if err != nil {
				t.Fatalf("読み直しでエラーを期待しなかったが %v だった", err)
			}
			if again.Body != got.Body {
				t.Errorf("読み直した本文 = %q、期待値 = %q", again.Body, got.Body)
			}
		})
	}
}

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		content    string
		wantTitle  *string
		wantNumber *int32
		wantETag   string
		wantBody   string
	}{
		{
			name:     "frontmatterの無いファイルは全体を本文にする",
			content:  "# 見出し\n\n本文\n",
			wantBody: "# 見出し\n\n本文\n",
		},
		{
			name:     "Wikinoのキーの無いfrontmatterは本文に残す",
			content:  "---\ntags: [a]\n---\n\n本文\n",
			wantBody: "---\ntags: [a]\n---\n\n本文\n",
		},
		{
			name:       "読み取り専用のキーは読まずに取り除く",
			content:    "---\nwikino_number: 12\nwikino_etag: \"\\\"v1\\\"\"\nwikino_status: \"published\"\nwikino_url: \"https://example.com\"\n---\n\n本文\n",
			wantNumber: ptr(int32(12)),
			wantETag:   `"v1"`,
			wantBody:   "本文\n",
		},
		{
			name:     "空行を消したファイルは区切りの直後から本文にする",
			content:  "---\nwikino_etag: \"v1\"\n---\n本文\n",
			wantETag: "v1",
			wantBody: "本文\n",
		},
		{
			name:     "空の本文",
			content:  "---\nwikino_etag: \"v1\"\n---\n",
			wantETag: "v1",
			wantBody: "",
		},
		{
			name:     "末尾に改行の無い閉じの区切り",
			content:  "---\nwikino_etag: \"v1\"\n---",
			wantETag: "v1",
			wantBody: "",
		},
		{
			name:      "引用符の無いタイトルも読む",
			content:   "---\nwikino_title: 議事録\nwikino_etag: v1\n---\n\n本文\n",
			wantTitle: ptr("議事録"),
			wantETag:  "v1",
			wantBody:  "本文\n",
		},
		{
			name:     "本文自身のfrontmatterは、編集されたキーを含めて残す",
			content:  "---\nwikino_etag: \"v1\"\n# 説明\ntags: [a, b, c]\n---\n本文\n",
			wantETag: "v1",
			wantBody: "---\n# 説明\ntags: [a, b, c]\n---\n本文\n",
		},
		{
			name:     "CRLFのファイル",
			content:  "---\r\nwikino_etag: \"v1\"\r\n---\r\n\r\n本文\r\n",
			wantETag: "v1",
			wantBody: "本文\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Parse(tt.content)
			if err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}
			if got.Body != tt.wantBody {
				t.Errorf("Body = %q、期待値 = %q", got.Body, tt.wantBody)
			}
			if got.ETag != tt.wantETag {
				t.Errorf("ETag = %q、期待値 = %q", got.ETag, tt.wantETag)
			}
			if (got.Title == nil) != (tt.wantTitle == nil) || (got.Title != nil && *got.Title != *tt.wantTitle) {
				t.Errorf("Title = %v、期待値 = %v", got.Title, tt.wantTitle)
			}
			if (got.Number == nil) != (tt.wantNumber == nil) || (got.Number != nil && *got.Number != *tt.wantNumber) {
				t.Errorf("Number = %v、期待値 = %v", got.Number, tt.wantNumber)
			}
		})
	}
}

func TestParse_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		wantKey string
	}{
		{name: "番号が文字列", content: "---\nwikino_number: \"12\"\n---\n", wantKey: "wikino_number"},
		{name: "番号が32ビットを超える", content: "---\nwikino_number: 4294967296\n---\n", wantKey: "wikino_number"},
		{name: "ETagが数値", content: "---\nwikino_etag: 123\n---\n", wantKey: "wikino_etag"},
		{name: "タイトルがnull", content: "---\nwikino_title: null\n---\n", wantKey: "wikino_title"},
		{name: "タイトルがリスト", content: "---\nwikino_title: [a]\ntags: []\n---\n", wantKey: "wikino_title"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := Parse(tt.content)
			var valueErr *InvalidValueError
			if !errors.As(err, &valueErr) || valueErr.Key != tt.wantKey {
				t.Errorf("エラー = %v、期待値 = %sのInvalidValueError", err, tt.wantKey)
			}
		})
	}

	t.Run("取り除くキーをエイリアスが参照する", func(t *testing.T) {
		t.Parallel()

		_, err := Parse("---\nwikino_etag: &v \"v1\"\naliases: [*v]\n---\n本文\n")
		if !errors.Is(err, ErrUnremovableKeys) {
			t.Errorf("エラー = %v、期待値 = ErrUnremovableKeys", err)
		}
	})
}

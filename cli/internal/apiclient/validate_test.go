package apiclient

import (
	"errors"
	"testing"
)

func assertReason(t *testing.T, err error, want InputErrorReason) {
	t.Helper()

	if want == "" {
		if err != nil {
			t.Errorf("受け付けるはずが失敗しました: %v", err)
		}
		return
	}

	var inputErr *InputError
	if !errors.As(err, &inputErr) {
		t.Fatalf("*InputErrorが返りません: %v", err)
	}
	if inputErr.Reason != want {
		t.Errorf("理由が違います: 実際 %q, 期待 %q", inputErr.Reason, want)
	}
}

func TestValidateHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
		// reasonが空なら受け付ける。
		reason InputErrorReason
	}{
		{name: "HTTPS", raw: "https://wikino.app", want: "https://wikino.app"},
		{name: "末尾のスラッシュ", raw: "https://wikino.app/", want: "https://wikino.app"},
		{name: "スキームの大文字", raw: "HTTPS://wikino.app", want: "https://wikino.app"},
		{name: "ポート", raw: "https://example.com:8443", want: "https://example.com:8443"},
		{name: "ループバックのHTTP (IPv4)", raw: "http://127.0.0.1:3000", want: "http://127.0.0.1:3000"},
		{name: "ループバックのHTTP (IPv6)", raw: "http://[::1]:3000", want: "http://[::1]:3000"},
		{name: "ループバックのHTTP (localhost)", raw: "http://localhost:3000", want: "http://localhost:3000"},
		{name: "ループバックでないHTTP", raw: "http://example.com", reason: ReasonInsecureScheme},
		{name: "ループバックに見せかけたホスト名", raw: "http://127.0.0.1.example.com", reason: ReasonInsecureScheme},
		{name: "ほかのスキーム", raw: "ftp://example.com", reason: ReasonInsecureScheme},
		{name: "スキームが無い", raw: "wikino.app", reason: ReasonInvalidFormat},
		{name: "パスがある", raw: "https://wikino.app/api/v1", reason: ReasonInvalidFormat},
		{name: "クエリがある", raw: "https://wikino.app?x=1", reason: ReasonInvalidFormat},
		{name: "フラグメントがある", raw: "https://wikino.app#x", reason: ReasonInvalidFormat},
		{name: "ユーザー情報がある", raw: "https://user:pass@wikino.app", reason: ReasonInvalidFormat},
		{name: "改行", raw: "https://wikino.app\n", reason: ReasonControlCharacter},
		{name: "空", raw: "", reason: ReasonEmpty},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			u, err := ValidateHost("--host", tt.raw)
			assertReason(t, err, tt.reason)
			if tt.reason == "" && u.String() != tt.want {
				t.Errorf("URLが違います: 実際 %q, 期待 %q", u.String(), tt.want)
			}
		})
	}
}

// エージェントが生成しがちな誤った値 (パストラバーサル・二重エンコード・制御文字・
// IDへのクエリの混入) を、要求を組み立てる前に具体的な理由で拒否することを確かめる。
func TestValidateSpaceIdentifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		value  string
		reason InputErrorReason
	}{
		{name: "英数字とハイフン", value: "my-space-01"},
		{name: "パストラバーサル", value: "../other", reason: ReasonDotSegment},
		{name: "ドット2つだけ", value: "..", reason: ReasonDotSegment},
		{name: "スラッシュ", value: "my-space/pages", reason: ReasonReservedCharacter},
		{name: "バックスラッシュ", value: `my-space\pages`, reason: ReasonReservedCharacter},
		{name: "エンコード済み", value: "my%2Dspace", reason: ReasonReservedCharacter},
		{name: "二重エンコード", value: "my%252Fspace", reason: ReasonReservedCharacter},
		{name: "クエリの混入", value: "my-space?limit=100", reason: ReasonReservedCharacter},
		{name: "フラグメントの混入", value: "my-space#top", reason: ReasonReservedCharacter},
		{name: "末尾の改行", value: "my-space\n", reason: ReasonControlCharacter},
		{name: "NUL", value: "my-space\x00", reason: ReasonControlCharacter},
		{name: "URLを丸ごと", value: "https://wikino.app/s/my-space", reason: ReasonReservedCharacter},
		{name: "空白", value: "my space", reason: ReasonInvalidFormat},
		{name: "全角文字", value: "スペース", reason: ReasonInvalidFormat},
		{name: "ドット", value: "my.space", reason: ReasonInvalidFormat},
		{name: "空", value: "", reason: ReasonEmpty},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertReason(t, ValidateSpaceIdentifier("--space", tt.value), tt.reason)
		})
	}
}

func TestParseNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		value  string
		want   int32
		reason InputErrorReason
	}{
		{name: "番号", value: "12", want: 12},
		{name: "上限", value: "2147483647", want: 2147483647},
		{name: "上限を超える", value: "2147483648", reason: ReasonInvalidFormat},
		{name: "0", value: "0", reason: ReasonInvalidFormat},
		{name: "負の数", value: "-1", reason: ReasonInvalidFormat},
		{name: "符号", value: "+12", reason: ReasonInvalidFormat},
		{name: "前後の空白", value: " 12", reason: ReasonInvalidFormat},
		{name: "全角の数字", value: "１２", reason: ReasonInvalidFormat},
		{name: "クエリの混入", value: "12?x=1", reason: ReasonReservedCharacter},
		{name: "フラグメントの混入", value: "12#body", reason: ReasonReservedCharacter},
		{name: "パスの混入", value: "12/draft_page", reason: ReasonReservedCharacter},
		{name: "パストラバーサル", value: "12/../13", reason: ReasonDotSegment},
		{name: "エンコード済み", value: "%31%32", reason: ReasonReservedCharacter},
		{name: "末尾の改行", value: "12\n", reason: ReasonControlCharacter},
		{name: "ページのURL", value: "https://wikino.app/s/my-space/pages/12", reason: ReasonReservedCharacter},
		{name: "空", value: "", reason: ReasonEmpty},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseNumber("<page_number>", tt.value)
			assertReason(t, err, tt.reason)
			if tt.reason == "" && got != tt.want {
				t.Errorf("番号が違います: 実際 %d, 期待 %d", got, tt.want)
			}
		})
	}
}

func TestParseUUID(t *testing.T) {
	t.Parallel()

	const id = "0b3a5c1e-6f43-4f6e-9a39-0f6f1b0f1a01"

	tests := []struct {
		name   string
		value  string
		reason InputErrorReason
	}{
		{name: "ハイフン区切りのUUID", value: id},
		{name: "大文字", value: "0B3A5C1E-6F43-4F6E-9A39-0F6F1B0F1A01"},
		{name: "ハイフンの無いUUID", value: "0b3a5c1e6f434f6e9a390f6f1b0f1a01", reason: ReasonInvalidFormat},
		{name: "波括弧", value: "{" + id + "}", reason: ReasonInvalidFormat},
		{name: "URN", value: "urn:uuid:" + id, reason: ReasonInvalidFormat},
		{name: "添付ファイルのパス", value: "/attachments/" + id, reason: ReasonReservedCharacter},
		{name: "クエリの混入", value: id + "?download=1", reason: ReasonReservedCharacter},
		{name: "パストラバーサル", value: "../" + id, reason: ReasonDotSegment},
		{name: "末尾の改行", value: id + "\n", reason: ReasonControlCharacter},
		{name: "途中で切れたUUID", value: id[:30], reason: ReasonInvalidFormat},
		{name: "空", value: "", reason: ReasonEmpty},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseUUID("<revision_id>", tt.value)
			assertReason(t, err, tt.reason)
			if tt.reason == "" && got.String() != id {
				t.Errorf("IDが違います: 実際 %q, 期待 %q", got.String(), id)
			}
		})
	}
}

func TestParseAPIPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
		// reasonが空なら受け付ける。
		reason InputErrorReason
	}{
		{name: "スペース配下のパス", raw: "/spaces/team/pages", want: "/spaces/team/pages"},
		{name: "先頭のスラッシュを省く", raw: "spaces/team/pages", want: "/spaces/team/pages"},
		{name: "クエリ", raw: "/spaces/team/pages?limit=5", want: "/spaces/team/pages?limit=5"},
		{name: "エンコードしていないクエリ", raw: "/spaces/team/pages?q=議事録 案", want: "/spaces/team/pages?q=%E8%AD%B0%E4%BA%8B%E9%8C%B2+%E6%A1%88"},
		{name: "エンコードしていないパス", raw: "/spaces/team/x y", want: "/spaces/team/x%20y"},
		{name: "ドットを含むセグメント", raw: "/spaces/team/a..b", want: "/spaces/team/a..b"},
		{name: "OpenAPI記述", raw: "/openapi.yaml", want: "/openapi.yaml"},
		{name: "空", raw: "", reason: ReasonEmpty},
		{name: "改行", raw: "/spaces/team\n/pages", reason: ReasonControlCharacter},
		{name: "URL", raw: "https://evil.example.com/spaces/team/pages", reason: ReasonNotAPIPath},
		{name: "スキームの無いホスト", raw: "//evil.example.com/spaces", reason: ReasonNotAPIPath},
		{name: "javascriptスキーム", raw: "javascript:alert(1)", reason: ReasonNotAPIPath},
		{name: "基底のパスを重ねる", raw: "/api/v1/spaces/team/pages", reason: ReasonNotAPIPath},
		{name: "基底のパスだけ", raw: "/api/v1", reason: ReasonNotAPIPath},
		{name: "フラグメント", raw: "/spaces/team/pages#x", reason: ReasonNotAPIPath},
		{name: "バックスラッシュ", raw: `/spaces\..\admin`, reason: ReasonNotAPIPath},
		{name: "誤ったエンコード", raw: "/spaces/%zz", reason: ReasonNotAPIPath},
		{name: "誤ったクエリ", raw: "/spaces/team/pages?q=%zz", reason: ReasonNotAPIPath},
		{name: "上の階層", raw: "/../oauth/token", reason: ReasonDotSegment},
		{name: "途中の上の階層", raw: "/spaces/team/../../oauth/token", reason: ReasonDotSegment},
		{name: "エンコードした上の階層", raw: "/spaces/%2e%2e/%2E%2E/oauth/token", reason: ReasonDotSegment},
		{name: "カレントのセグメント", raw: "/spaces/./team", reason: ReasonDotSegment},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ref, err := ParseAPIPath("<path>", tt.raw)
			assertReason(t, err, tt.reason)
			if tt.reason == "" && ref.String() != tt.want {
				t.Errorf("パスが違います: 実際 %q, 期待 %q", ref.String(), tt.want)
			}
		})
	}
}

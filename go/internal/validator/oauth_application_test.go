package validator_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestOAuthApplicationCreateValidator_Valid(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	v := validator.NewOAuthApplicationCreateValidator()

	tests := []struct {
		name             string
		appName          string
		redirectURIs     string
		clientType       string
		wantRedirectURIs []string
		wantClientType   model.OAuthClientType
	}{
		{
			name:             "HTTPSのURIを1つ登録するconfidentialクライアント",
			appName:          "Webアプリ",
			redirectURIs:     "https://example.com/callback",
			clientType:       "confidential",
			wantRedirectURIs: []string{"https://example.com/callback"},
			wantClientType:   model.OAuthClientTypeConfidential,
		},
		{
			// 認可要求と完全一致で照合するため、クエリや末尾のスラッシュを含めて書いた形のまま保存する。
			name:             "空行と前後の空白と重複を除き、URIそのものは正規化しない",
			appName:          strings.Repeat("あ", 50),
			redirectURIs:     "  https://example.com/cb/?a=1 \r\n\r\nhttps://example.com/cb/?a=1\nHTTPS://Example.com:8443/cb\n",
			clientType:       "public",
			wantRedirectURIs: []string{"https://example.com/cb/?a=1", "HTTPS://Example.com:8443/cb"},
			wantClientType:   model.OAuthClientTypePublic,
		},
		{
			name:             "ループバックのIPアドレスだけはHTTPを許す",
			appName:          "CLI",
			redirectURIs:     "http://127.0.0.1/callback\nhttp://127.0.0.1:8080/callback\nhttp://[::1]:8080/callback",
			clientType:       "public",
			wantRedirectURIs: []string{"http://127.0.0.1/callback", "http://127.0.0.1:8080/callback", "http://[::1]:8080/callback"},
			wantClientType:   model.OAuthClientTypePublic,
		},
		{
			name:             "リダイレクトURIを10個まで登録できる",
			appName:          "Webアプリ",
			redirectURIs:     redirectURILines(10),
			clientType:       "confidential",
			wantRedirectURIs: strings.Split(redirectURILines(10), "\n"),
			wantClientType:   model.OAuthClientTypeConfidential,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			output, err := v.Validate(ctx, validator.OAuthApplicationCreateValidatorInput{
				Name:         tt.appName,
				RedirectURIs: tt.redirectURIs,
				ClientType:   tt.clientType,
			})
			if err != nil {
				t.Fatalf("Validate()のエラー = %v", err)
			}
			if !slices.Equal(output.RedirectURIs, tt.wantRedirectURIs) {
				t.Errorf("RedirectURIs = %q、期待値 = %q", output.RedirectURIs, tt.wantRedirectURIs)
			}
			if output.ClientType != tt.wantClientType {
				t.Errorf("ClientType = %v、期待値 = %v", output.ClientType, tt.wantClientType)
			}
		})
	}
}

func TestOAuthApplicationCreateValidator_Invalid(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	v := validator.NewOAuthApplicationCreateValidator()

	valid := validator.OAuthApplicationCreateValidatorInput{
		Name:         "Webアプリ",
		RedirectURIs: "https://example.com/callback",
		ClientType:   "confidential",
	}

	redirectURIs := func(value string) func(*validator.OAuthApplicationCreateValidatorInput) {
		return func(i *validator.OAuthApplicationCreateValidatorInput) { i.RedirectURIs = value }
	}

	tests := []struct {
		name   string
		modify func(*validator.OAuthApplicationCreateValidatorInput)
		field  string
	}{
		{name: "名前が空", modify: func(i *validator.OAuthApplicationCreateValidatorInput) { i.Name = "" }, field: "name"},
		{name: "名前が50文字を超える", modify: func(i *validator.OAuthApplicationCreateValidatorInput) { i.Name = strings.Repeat("あ", 51) }, field: "name"},
		{name: "名前に改行を含む", modify: func(i *validator.OAuthApplicationCreateValidatorInput) { i.Name = "foo\nbar" }, field: "name"},
		{name: "名前の先頭が空白", modify: func(i *validator.OAuthApplicationCreateValidatorInput) { i.Name = " Webアプリ" }, field: "name"},
		{name: "リダイレクトURIが空", modify: redirectURIs(""), field: "redirect_uris"},
		{name: "リダイレクトURIが空行だけ", modify: redirectURIs("\n \n"), field: "redirect_uris"},
		{name: "リダイレクトURIが10個を超える", modify: redirectURIs(redirectURILines(11)), field: "redirect_uris"},
		{name: "リダイレクトURIが2000文字を超える", modify: redirectURIs("https://example.com/" + strings.Repeat("a", 1981)), field: "redirect_uris"},
		{name: "HTTPのURI", modify: redirectURIs("http://example.com/callback"), field: "redirect_uris"},
		{name: "localhostへのHTTPのURI", modify: redirectURIs("http://localhost:8080/callback"), field: "redirect_uris"},
		{name: "127.0.0.1以外のループバックへのHTTPのURI", modify: redirectURIs("http://127.0.0.2/callback"), field: "redirect_uris"},
		{name: "カスタムスキームのURI", modify: redirectURIs("com.example.app:/callback"), field: "redirect_uris"},
		{name: "javascriptスキームのURI", modify: redirectURIs("javascript:alert(1)"), field: "redirect_uris"},
		{name: "相対URI", modify: redirectURIs("/callback"), field: "redirect_uris"},
		{name: "ホストの無いURI", modify: redirectURIs("https:///callback"), field: "redirect_uris"},
		{name: "フラグメントを含むURI", modify: redirectURIs("https://example.com/callback#done"), field: "redirect_uris"},
		{name: "空のフラグメントを含むURI", modify: redirectURIs("https://example.com/callback#"), field: "redirect_uris"},
		{name: "ユーザー情報を含むURI", modify: redirectURIs("https://user:pass@example.com/callback"), field: "redirect_uris"},
		{name: "途中に空白を含むURI", modify: redirectURIs("https://example.com/call back"), field: "redirect_uris"},
		{name: "正しいURIと不正なURIが混ざっている", modify: redirectURIs("https://example.com/callback\nhttp://example.com/callback"), field: "redirect_uris"},
		{name: "クライアントの種別が空", modify: func(i *validator.OAuthApplicationCreateValidatorInput) { i.ClientType = "" }, field: "client_type"},
		{name: "クライアントの種別が選択肢に無い", modify: func(i *validator.OAuthApplicationCreateValidatorInput) { i.ClientType = "native" }, field: "client_type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := valid
			tt.modify(&input)

			output, err := v.Validate(ctx, input)
			if output != nil {
				t.Errorf("Validate()の出力 = %v、期待値 = nil", output)
			}
			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatalf("Validate()のエラー = %v、期待値 = ValidationError", err)
			}
			if !ve.HasFieldError(tt.field) {
				t.Errorf("%sのフィールドエラーが無い", tt.field)
			}
		})
	}
}

// 1行に複数の問題があっても、どのURIが受け付けられないかを利用者が見つけられるよう、
// URIごとにエラーを示す。
func TestOAuthApplicationCreateValidator_URIごとにエラーを示す(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	v := validator.NewOAuthApplicationCreateValidator()

	_, err := v.Validate(ctx, validator.OAuthApplicationCreateValidatorInput{
		Name:         "Webアプリ",
		RedirectURIs: "http://example.com/a\nhttps://example.com/ok\nhttps://example.com/b#x",
		ClientType:   "confidential",
	})
	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatalf("Validate()のエラー = %v、期待値 = ValidationError", err)
	}

	got := ve.GetFieldErrors("redirect_uris")
	if len(got) != 2 {
		t.Fatalf("redirect_urisのエラー = %q、期待値は2件", got)
	}
	if !strings.Contains(got[0], "http://example.com/a") || !strings.Contains(got[1], "https://example.com/b#x") {
		t.Errorf("redirect_urisのエラー = %q、それぞれのURIを含むこと", got)
	}
}

// redirectURILinesは、異なるHTTPSのリダイレクトURIをn行並べたテキストを返す。
func redirectURILines(n int) string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("https://example.com/callback/%d", i)
	}
	return strings.Join(lines, "\n")
}

// 名前とリダイレクトURIの規則は登録と共有しているため、境界の値は登録のテストで押さえ、
// ここでは編集が同じ規則で検証して変換することだけを確かめる。
func TestOAuthApplicationUpdateValidator(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	v := validator.NewOAuthApplicationUpdateValidator()

	t.Run("空行と重複を除いたリダイレクトURIを返す", func(t *testing.T) {
		t.Parallel()

		output, err := v.Validate(ctx, validator.OAuthApplicationUpdateValidatorInput{
			Name:         "編集したアプリ",
			RedirectURIs: "https://example.com/callback\n\nhttps://example.com/callback\nhttp://[::1]:8080/callback",
		})
		if err != nil {
			t.Fatalf("Validate()のエラー = %v", err)
		}
		if want := []string{"https://example.com/callback", "http://[::1]:8080/callback"}; !slices.Equal(output.RedirectURIs, want) {
			t.Errorf("RedirectURIs = %q、期待値 = %q", output.RedirectURIs, want)
		}
	})

	t.Run("名前とリダイレクトURIの誤りを項目ごとに返す", func(t *testing.T) {
		t.Parallel()

		_, err := v.Validate(ctx, validator.OAuthApplicationUpdateValidatorInput{
			Name:         "",
			RedirectURIs: "http://example.com/callback",
		})
		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatalf("Validate()のエラー = %v、期待値はValidationError", err)
		}
		if !ve.HasFieldError("name") {
			t.Error("nameのエラーが無い")
		}
		if !ve.HasFieldError("redirect_uris") {
			t.Error("redirect_urisのエラーが無い")
		}
		if ve.HasFieldError("client_type") {
			t.Error("編集で受け取らないclient_typeのエラーがある")
		}
	})
}

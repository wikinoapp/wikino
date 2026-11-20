package auth

import (
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

const testTOTPSecret = "JBSWY3DPEHPK3PXPJBSWY3DPEHPK3PXP"

// totpCodeAtはat時点の正しいコードを返す
func totpCodeAt(t *testing.T, at time.Time) string {
	t.Helper()

	code, err := totp.GenerateCodeCustom(testTOTPSecret, at, totpOpts)
	if err != nil {
		t.Fatalf("コードの生成に失敗: %v", err)
	}
	return code
}

func TestGenerateTOTPSecret(t *testing.T) {
	t.Parallel()

	first, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("GenerateTOTPSecret()のエラー = %v", err)
	}
	if !regexp.MustCompile(`^[A-Z2-7]{32}$`).MatchString(first) {
		t.Errorf("秘密鍵 = %q、32文字のbase32を期待", first)
	}

	second, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("GenerateTOTPSecret()のエラー = %v", err)
	}
	if first == second {
		t.Error("2回生成した秘密鍵が一致した")
	}

	// 生成した秘密鍵でコードを作り、照合が通ること
	now := time.Now()
	code, err := totp.GenerateCodeCustom(first, now, totpOpts)
	if err != nil {
		t.Fatalf("コードの生成に失敗: %v", err)
	}
	if !ValidateTOTPCode(code, first, now) {
		t.Error("生成した秘密鍵のコードが照合で通らない")
	}
}

func TestBuildOTPAuthURL(t *testing.T) {
	t.Parallel()

	raw, err := BuildOTPAuthURL(testTOTPSecret, "user@example.com")
	if err != nil {
		t.Fatalf("BuildOTPAuthURL()のエラー = %v", err)
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("URIの解析に失敗: %v", err)
	}
	if u.Scheme != "otpauth" || u.Host != "totp" {
		t.Errorf("スキームとホスト = %q・%q、期待値 = otpauth・totp", u.Scheme, u.Host)
	}
	if u.Path != "/Wikino:user@example.com" {
		t.Errorf("パス = %q、期待値 = %q", u.Path, "/Wikino:user@example.com")
	}
	q := u.Query()
	if q.Get("secret") != testTOTPSecret {
		t.Errorf("secret = %q、期待値 = %q", q.Get("secret"), testTOTPSecret)
	}
	if q.Get("issuer") != "Wikino" {
		t.Errorf("issuer = %q、期待値 = %q", q.Get("issuer"), "Wikino")
	}
}

func TestBuildOTPAuthURL_読めない秘密鍵(t *testing.T) {
	t.Parallel()

	if _, err := BuildOTPAuthURL("not base32!", "user@example.com"); err == nil {
		t.Error("エラーを期待したが、nilだった")
	}
}

func TestValidateTOTPCode(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 14, 12, 0, 10, 0, time.UTC)

	testCases := []struct {
		name string
		code string
		want bool
	}{
		{name: "今のコード", code: totpCodeAt(t, now), want: true},
		{name: "1つ前のタイムステップのコード", code: totpCodeAt(t, now.Add(-30*time.Second)), want: true},
		{name: "1つ後のタイムステップのコード", code: totpCodeAt(t, now.Add(30*time.Second)), want: true},
		{name: "前後15秒のずれを足した範囲のコード", code: totpCodeAt(t, now.Add(-45*time.Second)), want: true},
		{name: "2分前のコード", code: totpCodeAt(t, now.Add(-2*time.Minute)), want: false},
		{name: "6桁の数字でない", code: "abcdef", want: false},
		{name: "空", code: "", want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := ValidateTOTPCode(tc.code, testTOTPSecret, now); got != tc.want {
				t.Errorf("ValidateTOTPCode() = %v、期待値 = %v", got, tc.want)
			}
		})
	}
}

func TestValidateTOTPCode_読めない秘密鍵(t *testing.T) {
	t.Parallel()

	if ValidateTOTPCode("123456", "not base32!", time.Now()) {
		t.Error("読めない秘密鍵で照合が通った")
	}
}

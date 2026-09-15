package image

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestNewHelper_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		baseURL string
		key     string
		salt    string
		wantErr bool
	}{
		{
			name:    "正常系: 16進数でkeyとsaltを指定",
			baseURL: "https://imgproxy.example.dev",
			key:     "deadbeef",
			salt:    "cafef00d",
			wantErr: false,
		},
		{
			name:    "正常系: 大文字16進数も受け入れる",
			baseURL: "https://imgproxy.example.dev",
			key:     "DEADBEEF",
			salt:    "CAFEF00D",
			wantErr: false,
		},
		{
			name:    "正常系: 空のkey/salt (signing無効モード相当)",
			baseURL: "https://imgproxy.example.dev",
			key:     "",
			salt:    "",
			wantErr: false,
		},
		{
			name:    "異常系: baseURLが空",
			baseURL: "",
			key:     "deadbeef",
			salt:    "cafef00d",
			wantErr: true,
		},
		{
			name:    "異常系: keyが奇数長",
			baseURL: "https://imgproxy.example.dev",
			key:     "abc",
			salt:    "cafef00d",
			wantErr: true,
		},
		{
			name:    "異常系: saltに16進数以外の文字",
			baseURL: "https://imgproxy.example.dev",
			key:     "deadbeef",
			salt:    "zzzzzzzz",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h, err := NewHelper(tt.baseURL, tt.key, tt.salt)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("エラーを期待したがnil")
				}
				return
			}
			if err != nil {
				t.Fatalf("予期しないエラー: %v", err)
			}
			if h == nil {
				t.Fatal("Helperがnil")
			}
		})
	}
}

func TestHelper_BuildURL(t *testing.T) {
	t.Parallel()

	const (
		baseURL = "https://imgproxy.example.dev"
		keyHex  = "1a2b3c4d5e6f"
		saltHex = "0f1e2d3c4b5a"
	)

	helper, err := NewHelper(baseURL, keyHex, saltHex)
	if err != nil {
		t.Fatalf("NewHelperでエラー: %v", err)
	}

	expiresAt := time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		sourceURL string
		opts      ResizeOptions
		wantSubs  []string // URL内に含まれるべき文字列
		wantErr   bool
	}{
		{
			name:      "正常系: og用1200x630, format auto, expires付き",
			sourceURL: "s3://my-bucket/path/to/key",
			opts: ResizeOptions{
				Width:     1200,
				Height:    630,
				Format:    "auto",
				ExpiresAt: expiresAt,
			},
			wantSubs: []string{
				baseURL + "/",
				"resize:fit:1200:630",
				"format:auto",
				"plain/s3://my-bucket/path/to/key",
				"expires:" + formatUnix(expiresAt),
			},
		},
		{
			name:      "正常系: format未指定ならformatオプションを付けない",
			sourceURL: "s3://my-bucket/path/to/key",
			opts: ResizeOptions{
				Width:  600,
				Height: 600,
			},
			wantSubs: []string{
				"resize:fit:600:600",
				"plain/s3://my-bucket/path/to/key",
			},
		},
		{
			name:      "正常系: ExpiresAtがゼロ値ならexpiresオプションを付けない",
			sourceURL: "s3://my-bucket/key",
			opts: ResizeOptions{
				Width:  100,
				Height: 100,
				Format: "webp",
			},
			wantSubs: []string{
				"resize:fit:100:100",
				"format:webp",
				"plain/s3://my-bucket/key",
			},
		},
		{
			name:      "異常系: sourceURLが空",
			sourceURL: "",
			opts:      ResizeOptions{Width: 100, Height: 100},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := helper.BuildURL(tt.sourceURL, tt.opts)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("エラーを期待したが、nilだった: %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("予期しないエラー: %v", err)
			}

			for _, sub := range tt.wantSubs {
				if !strings.Contains(got, sub) {
					t.Errorf("URLに%qが含まれていない: %q", sub, got)
				}
			}

			// expires未指定のテストではexpiresが含まれていないことを検証
			if tt.opts.ExpiresAt.IsZero() && strings.Contains(got, "expires:") {
				t.Errorf("ExpiresAt未指定なのにexpiresが含まれている: %q", got)
			}
			if tt.opts.Format == "" && strings.Contains(got, "format:") {
				t.Errorf("Format未指定なのにformatが含まれている: %q", got)
			}
		})
	}
}

func TestHelper_BuildURL_SignatureDeterministic(t *testing.T) {
	t.Parallel()

	helper, err := NewHelper("https://imgproxy.example.dev", "deadbeef", "cafef00d")
	if err != nil {
		t.Fatalf("NewHelperでエラー: %v", err)
	}

	opts := ResizeOptions{Width: 100, Height: 100, Format: "auto"}

	// 同じ入力で2回呼び出すと完全一致すること (HMAC-SHA256は決定的)
	first, err := helper.BuildURL("s3://b/k", opts)
	if err != nil {
		t.Fatalf("BuildURLでエラー: %v", err)
	}
	second, err := helper.BuildURL("s3://b/k", opts)
	if err != nil {
		t.Fatalf("BuildURLでエラー: %v", err)
	}
	if first != second {
		t.Errorf("同一入力でURLが異なる:\n  first=%q\n  second=%q", first, second)
	}

	// salt / keyが違えば署名部分が変わること
	helper2, err := NewHelper("https://imgproxy.example.dev", "deadbeef", "11111111")
	if err != nil {
		t.Fatalf("NewHelperでエラー: %v", err)
	}
	other, err := helper2.BuildURL("s3://b/k", opts)
	if err != nil {
		t.Fatalf("BuildURLでエラー: %v", err)
	}
	if first == other {
		t.Errorf("saltが異なるのに同じURLになった: %q", first)
	}
}

func TestHelper_BuildURL_SignatureFormat(t *testing.T) {
	t.Parallel()

	// imgproxy v3のURL構造: {baseURL}/{base64-url-safe-signature}/{processing}/.../plain/{source}
	helper, err := NewHelper("https://imgproxy.example.dev", "deadbeef", "cafef00d")
	if err != nil {
		t.Fatalf("NewHelperでエラー: %v", err)
	}

	got, err := helper.BuildURL("s3://b/k", ResizeOptions{Width: 1, Height: 1})
	if err != nil {
		t.Fatalf("BuildURLでエラー: %v", err)
	}

	prefix := "https://imgproxy.example.dev/"
	if !strings.HasPrefix(got, prefix) {
		t.Fatalf("URLがbaseURLで始まっていない: %q", got)
	}

	// 署名部分は最初のスラッシュ以降、次のスラッシュまで
	rest := strings.TrimPrefix(got, prefix)
	idx := strings.Index(rest, "/")
	if idx <= 0 {
		t.Fatalf("署名部分が見つからない: %q", got)
	}
	signature := rest[:idx]

	// HMAC-SHA256をbase64 url-safe (パディングなし) で表現すると43文字
	if len(signature) != 43 {
		t.Errorf("署名長 = %d (%q)、期待値 = 43", len(signature), signature)
	}
	// '+' '/' '=' を含まない (URL-safeであること)
	if strings.ContainsAny(signature, "+/=") {
		t.Errorf("署名がURL-safeではない文字を含む: %q", signature)
	}
}

// formatUnixはtime.Timeをimgproxyのexpiresオプションで使うUnixタイムスタンプ文字列に変換する
func formatUnix(t time.Time) string {
	return strconv.FormatInt(t.Unix(), 10)
}

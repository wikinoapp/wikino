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
			name:    "正常系: 16進数で key と salt を指定",
			baseURL: "https://imgproxy.example.dev",
			key:     "deadbeef",
			salt:    "cafef00d",
			wantErr: false,
		},
		{
			name:    "正常系: 大文字 16 進数も受け入れる",
			baseURL: "https://imgproxy.example.dev",
			key:     "DEADBEEF",
			salt:    "CAFEF00D",
			wantErr: false,
		},
		{
			name:    "正常系: 空の key/salt (signing 無効モード相当)",
			baseURL: "https://imgproxy.example.dev",
			key:     "",
			salt:    "",
			wantErr: false,
		},
		{
			name:    "異常系: baseURL が空",
			baseURL: "",
			key:     "deadbeef",
			salt:    "cafef00d",
			wantErr: true,
		},
		{
			name:    "異常系: key が奇数長",
			baseURL: "https://imgproxy.example.dev",
			key:     "abc",
			salt:    "cafef00d",
			wantErr: true,
		},
		{
			name:    "異常系: salt に 16 進数以外の文字",
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
					t.Fatalf("エラーを期待したが nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("予期しないエラー: %v", err)
			}
			if h == nil {
				t.Fatal("Helper が nil")
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
		t.Fatalf("NewHelper でエラー: %v", err)
	}

	expiresAt := time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		sourceURL string
		opts      ResizeOptions
		wantSubs  []string // URL 内に含まれるべき文字列
		wantErr   bool
	}{
		{
			name:      "正常系: og 用 1200x630, format auto, expires 付き",
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
			name:      "正常系: format 未指定なら format オプションを付けない",
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
			name:      "正常系: ExpiresAt がゼロ値なら expires オプションを付けない",
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
			name:      "異常系: sourceURL が空",
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
					t.Fatalf("エラーを期待したが nil。got=%q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("予期しないエラー: %v", err)
			}

			for _, sub := range tt.wantSubs {
				if !strings.Contains(got, sub) {
					t.Errorf("URL に %q が含まれていない: got=%q", sub, got)
				}
			}

			// expires 未指定のテストでは expires が含まれていないことを検証
			if tt.opts.ExpiresAt.IsZero() && strings.Contains(got, "expires:") {
				t.Errorf("ExpiresAt 未指定なのに expires が含まれている: got=%q", got)
			}
			if tt.opts.Format == "" && strings.Contains(got, "format:") {
				t.Errorf("Format 未指定なのに format が含まれている: got=%q", got)
			}
		})
	}
}

func TestHelper_BuildURL_SignatureDeterministic(t *testing.T) {
	t.Parallel()

	helper, err := NewHelper("https://imgproxy.example.dev", "deadbeef", "cafef00d")
	if err != nil {
		t.Fatalf("NewHelper でエラー: %v", err)
	}

	opts := ResizeOptions{Width: 100, Height: 100, Format: "auto"}

	// 同じ入力で 2 回呼び出すと完全一致すること (HMAC-SHA256 は決定的)
	first, err := helper.BuildURL("s3://b/k", opts)
	if err != nil {
		t.Fatalf("BuildURL でエラー: %v", err)
	}
	second, err := helper.BuildURL("s3://b/k", opts)
	if err != nil {
		t.Fatalf("BuildURL でエラー: %v", err)
	}
	if first != second {
		t.Errorf("同一入力で URL が異なる:\n  first=%q\n  second=%q", first, second)
	}

	// salt / key が違えば署名部分が変わること
	helper2, err := NewHelper("https://imgproxy.example.dev", "deadbeef", "11111111")
	if err != nil {
		t.Fatalf("NewHelper でエラー: %v", err)
	}
	other, err := helper2.BuildURL("s3://b/k", opts)
	if err != nil {
		t.Fatalf("BuildURL でエラー: %v", err)
	}
	if first == other {
		t.Errorf("salt が異なるのに同じ URL になった: %q", first)
	}
}

func TestHelper_BuildURL_SignatureFormat(t *testing.T) {
	t.Parallel()

	// imgproxy v3 の URL 構造: {baseURL}/{base64-url-safe-signature}/{processing}/.../plain/{source}
	helper, err := NewHelper("https://imgproxy.example.dev", "deadbeef", "cafef00d")
	if err != nil {
		t.Fatalf("NewHelper でエラー: %v", err)
	}

	got, err := helper.BuildURL("s3://b/k", ResizeOptions{Width: 1, Height: 1})
	if err != nil {
		t.Fatalf("BuildURL でエラー: %v", err)
	}

	prefix := "https://imgproxy.example.dev/"
	if !strings.HasPrefix(got, prefix) {
		t.Fatalf("URL が baseURL で始まっていない: %q", got)
	}

	// 署名部分は最初のスラッシュ以降、次のスラッシュまで
	rest := strings.TrimPrefix(got, prefix)
	idx := strings.Index(rest, "/")
	if idx <= 0 {
		t.Fatalf("署名部分が見つからない: %q", got)
	}
	signature := rest[:idx]

	// HMAC-SHA256 を base64 url-safe (パディングなし) で表現すると 43 文字
	if len(signature) != 43 {
		t.Errorf("署名長が想定と異なる: got=%d (%q), want=43", len(signature), signature)
	}
	// '+' '/' '=' を含まない (URL-safe であること)
	if strings.ContainsAny(signature, "+/=") {
		t.Errorf("署名が URL-safe ではない文字を含む: %q", signature)
	}
}

// formatUnix は time.Time を imgproxy の expires オプションで使う Unix タイムスタンプ文字列に変換する
func formatUnix(t time.Time) string {
	return strconv.FormatInt(t.Unix(), 10)
}

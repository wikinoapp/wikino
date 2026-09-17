package image

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestNewOgImageBuilder_Errors(t *testing.T) {
	t.Parallel()

	helper, err := NewHelper("https://imgproxy.example.dev", "deadbeef", "cafef00d")
	if err != nil {
		t.Fatalf("NewHelperでエラー: %v", err)
	}

	tests := []struct {
		name    string
		helper  *Helper
		bucket  string
		wantErr bool
	}{
		{
			name:    "正常系: helperとbucketの両方を指定",
			helper:  helper,
			bucket:  "my-bucket",
			wantErr: false,
		},
		{
			name:    "異常系: helperがnil",
			helper:  nil,
			bucket:  "my-bucket",
			wantErr: true,
		},
		{
			name:    "異常系: bucketが空",
			helper:  helper,
			bucket:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b, err := NewOgImageBuilder(tt.helper, tt.bucket)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("エラーを期待したがnil")
				}
				return
			}
			if err != nil {
				t.Fatalf("予期しないエラー: %v", err)
			}
			if b == nil {
				t.Fatal("OgImageBuilderがnil")
			}
		})
	}
}

func TestOgImageBuilder_BuildOgImageURL(t *testing.T) {
	t.Parallel()

	helper, err := NewHelper("https://imgproxy.example.dev", "deadbeef", "cafef00d")
	if err != nil {
		t.Fatalf("NewHelperでエラー: %v", err)
	}
	builder, err := NewOgImageBuilder(helper, "my-bucket")
	if err != nil {
		t.Fatalf("NewOgImageBuilderでエラー: %v", err)
	}

	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)

	t.Run("正常系: 1200x630 / format jpg / 1時間expiresが組み込まれる", func(t *testing.T) {
		t.Parallel()

		got, err := builder.BuildOgImageURL("path/to/blob.png", now)
		if err != nil {
			t.Fatalf("BuildOgImageURLでエラー: %v", err)
		}

		// og:imageのポリシーがBuilder内部に集約されていることをURL構造で検証する
		wantSubs := []string{
			"https://imgproxy.example.dev/",
			"resize:fit:1200:630",
			"format:jpg",
			"plain/s3://my-bucket/path/to/blob.png",
			"expires:" + strconv.FormatInt(now.Add(time.Hour).Unix(), 10),
		}
		for _, sub := range wantSubs {
			if !strings.Contains(got, sub) {
				t.Errorf("URLに%qが含まれていない: %q", sub, got)
			}
		}
	})

	t.Run("異常系: blobKeyが空", func(t *testing.T) {
		t.Parallel()

		_, err := builder.BuildOgImageURL("", now)
		if err == nil {
			t.Fatal("blobKeyが空のときはエラーを期待したがnil")
		}
	})

	t.Run("正常系: nowが変わるとexpiresも変わる (TTLが固定であることの検証)", func(t *testing.T) {
		t.Parallel()

		later := now.Add(2 * time.Hour)
		first, err := builder.BuildOgImageURL("k", now)
		if err != nil {
			t.Fatalf("BuildOgImageURLでエラー: %v", err)
		}
		second, err := builder.BuildOgImageURL("k", later)
		if err != nil {
			t.Fatalf("BuildOgImageURLでエラー: %v", err)
		}
		if !strings.Contains(first, "expires:"+strconv.FormatInt(now.Add(time.Hour).Unix(), 10)) {
			t.Errorf("1つ目のexpiresが想定と異なる: %q", first)
		}
		if !strings.Contains(second, "expires:"+strconv.FormatInt(later.Add(time.Hour).Unix(), 10)) {
			t.Errorf("2つ目のexpiresが想定と異なる: %q", second)
		}
	})
}

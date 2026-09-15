package email

import (
	"bytes"
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestExportSender_SendSucceeded(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		locale      string
		wantInBody  string
		wantSubject string
	}{
		{
			name:        "日本語",
			locale:      "ja",
			wantInBody:  "エクスポートデータの用意ができました。",
			wantSubject: "[Wikino] エクスポートデータの用意ができました",
		},
		{
			name:        "英語",
			locale:      "en",
			wantInBody:  "Your export data is ready.",
			wantSubject: "[Wikino] Your export data is ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			noop := NewNoopSender()
			sender := NewExportSender(noop)

			ctx := i18n.SetLocale(context.Background(), tt.locale)
			downloadURL := "https://example.dev/s/wikino/settings/exports/abc/download"
			if err := sender.SendSucceeded(ctx, "test@example.com", downloadURL, "https://example.dev", tt.locale); err != nil {
				t.Fatalf("SendSucceeded()のエラー = %v", err)
			}

			if len(noop.SentEmails) != 1 {
				t.Fatalf("SentEmailsの件数 = %d、期待値 = 1", len(noop.SentEmails))
			}

			sent := noop.SentEmails[0]
			if sent.To != "test@example.com" {
				t.Errorf("To = %s、期待値 = test@example.com", sent.To)
			}
			if sent.Subject != tt.wantSubject {
				t.Errorf("Subject = %q、期待値 = %q", sent.Subject, tt.wantSubject)
			}

			body := renderComponent(t, ctx, sent.TextBody)
			if !strings.Contains(body, tt.wantInBody) {
				t.Errorf("本文に%qが含まれていません: %q", tt.wantInBody, body)
			}
			if !strings.Contains(body, downloadURL) {
				t.Errorf("本文にダウンロードURLが含まれていません: %q", body)
			}

			// 文面はダウンロードのリンクが期限切れになるのと同じ定数から組み立てるため、
			// 両者が別のことを言うことはない。
			hours := int(model.ExportDownloadExpiration.Hours())
			if !strings.Contains(body, strconv.Itoa(hours)) {
				t.Errorf("本文に有効期間%dが含まれていません: %q", hours, body)
			}
		})
	}
}

func TestExportSender_SendFailed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		locale      string
		wantInBody  string
		wantSubject string
	}{
		{
			name:        "日本語",
			locale:      "ja",
			wantInBody:  "エクスポートデータを作成できませんでした。",
			wantSubject: "[Wikino] エクスポートデータを作成できませんでした",
		},
		{
			name:        "英語",
			locale:      "en",
			wantInBody:  "Your export could not be produced.",
			wantSubject: "[Wikino] Your export could not be produced",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			noop := NewNoopSender()
			sender := NewExportSender(noop)

			ctx := i18n.SetLocale(context.Background(), tt.locale)
			exportURL := "https://example.dev/s/wikino/settings/exports/abc"
			if err := sender.SendFailed(ctx, "test@example.com", exportURL, "https://example.dev", tt.locale); err != nil {
				t.Fatalf("SendFailed()のエラー = %v", err)
			}

			if len(noop.SentEmails) != 1 {
				t.Fatalf("SentEmailsの件数 = %d、期待値 = 1", len(noop.SentEmails))
			}

			sent := noop.SentEmails[0]
			if sent.Subject != tt.wantSubject {
				t.Errorf("Subject = %q、期待値 = %q", sent.Subject, tt.wantSubject)
			}

			body := renderComponent(t, ctx, sent.TextBody)
			if !strings.Contains(body, tt.wantInBody) {
				t.Errorf("本文に%qが含まれていません: %q", tt.wantInBody, body)
			}
			if !strings.Contains(body, exportURL) {
				t.Errorf("本文にエクスポート画面のURLが含まれていません: %q", body)
			}
		})
	}
}

// renderComponentはメール本文をレンダリングし、読み手が受け取る文面をテストが見られる
// ようにする。
func renderComponent(t *testing.T, ctx context.Context, component templ.Component) string {
	t.Helper()

	if component == nil {
		t.Fatal("テンプレートがnilです")
	}

	var buf bytes.Buffer
	if err := component.Render(ctx, &buf); err != nil {
		t.Fatalf("テンプレートのレンダリングに失敗: %v", err)
	}
	return buf.String()
}

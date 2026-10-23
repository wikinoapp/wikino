package cmdutil

import (
	"context"
	"errors"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/i18n"
)

func TestFactoryUsesDefaultHTTPClient(t *testing.T) {
	t.Parallel()

	f := &Factory{Localizer: i18n.NewLocalizer(i18n.LangJa)}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// クライアントを注入しない実行でも、APIとOAuthの両方が既定のクライアントへ委ねる。
	// 中止済みのcontextで、外部へ通信せずに要求の経路を確かめる。
	client, err := f.NewAPIClient("https://wikino.app", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetCurrentSpaceMemberWithResponse(ctx, "team"); !errors.Is(err, context.Canceled) {
		t.Errorf("APIのエラー = %v、期待値 = context.Canceled", err)
	}
	if _, err := f.OAuthClient("https://wikino.app").FetchMetadata(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("OAuthのエラー = %v、期待値 = context.Canceled", err)
	}
}

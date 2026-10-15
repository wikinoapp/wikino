package auth

import (
	"fmt"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

func TestConvertOAuthErrorLoginTimeout(t *testing.T) {
	t.Parallel()

	// ログインの待ち時間の超過は、拒否や不正な応答と分け、エージェントがやり直せるようにする。
	err := ConvertOAuthError(i18n.NewLocalizer(i18n.LangJa), "team", fmt.Errorf("ログイン: %w", ErrLoopbackTimeout))

	cliErr, ok := err.(*output.Error)
	if !ok {
		t.Fatalf("エラー = %v、期待値 = *output.Error", err)
	}
	if cliErr.Code != output.CodeLoginTimeout || cliErr.ExitCode != output.ExitError {
		t.Errorf("コード = %q、終了コード = %d、期待値 = %q, %d", cliErr.Code, cliErr.ExitCode, output.CodeLoginTimeout, output.ExitError)
	}
	if !strings.Contains(cliErr.Hint, "wikino auth login team") {
		t.Errorf("ヒント = %q", cliErr.Hint)
	}
}

func TestConvertOAuthErrorDeviceUnsupported(t *testing.T) {
	t.Parallel()

	// 接続先がWikinoでないことを疑わせるresponse_errorではなく、oauth_errorにして別の手段を案内する。
	err := ConvertOAuthError(i18n.NewLocalizer(i18n.LangJa), "team", fmt.Errorf("ログイン: %w", ErrDeviceAuthorizationUnsupported))

	cliErr, ok := err.(*output.Error)
	if !ok {
		t.Fatalf("エラー = %v、期待値 = *output.Error", err)
	}
	if cliErr.Code != output.CodeOAuthError || cliErr.ExitCode != output.ExitAuth {
		t.Errorf("コード = %q、終了コード = %d、期待値 = %q, %d", cliErr.Code, cliErr.ExitCode, output.CodeOAuthError, output.ExitAuth)
	}
	if !strings.Contains(cliErr.Message, "デバイス認可に対応していません") || !strings.Contains(cliErr.Hint, "--with-token") {
		t.Errorf("メッセージ = %q、ヒント = %q", cliErr.Message, cliErr.Hint)
	}
}

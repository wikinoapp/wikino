package usecase

import (
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/auth"
)

// TwoFactorAuthSetupは、認証アプリへの登録の途中で画面に出す秘密鍵
type TwoFactorAuthSetup struct {
	// Secretはbase32の秘密鍵。QRコードを読み取れないときに、認証アプリへ手で入力してもらう
	Secret string
	// OTPAuthURLは秘密鍵を認証アプリへ登録するotpauth URI。QRコードにする
	OTPAuthURL string
}

// newTwoFactorAuthSetupは秘密鍵から、画面に出すTwoFactorAuthSetupを作る。
// 認証アプリにはメールアドレスを表示し、どのアカウントのコードかを見分けられるようにする。
func newTwoFactorAuthSetup(secret, email string) (*TwoFactorAuthSetup, error) {
	url, err := auth.BuildOTPAuthURL(secret, email)
	if err != nil {
		return nil, fmt.Errorf("otpauth URIの作成に失敗: %w", err)
	}

	return &TwoFactorAuthSetup{Secret: secret, OTPAuthURL: url}, nil
}

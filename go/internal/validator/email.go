package validator

import "net/mail"

// IsValidEmailは、メールアドレスがアカウントのサインインとメール確認で
// 受理される形式かを返す。
func IsValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)

	return err == nil
}

// CanonicalEmailはメールアドレスのアドレス部分と、書かれた文字列がその
// アドレスだけであるかどうかを返す。まったく解釈できない場合は空のアドレスを返す。
//
// mail.ParseAddressは受理する値を正規化する。前後の空白や表示名はエラー無く
// 解釈され、返るアドレスは書かれた文字列と食い違う。解釈したアドレスではなく
// 渡された文字列を保存する呼び出し元は、この違いを知る必要がある。保存した文字列
// こそが、後の引き当てで一致させる対象になるため。
func CanonicalEmail(email string) (string, bool) {
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return "", false
	}

	return addr.Address, addr.Address == email
}

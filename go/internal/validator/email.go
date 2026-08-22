package validator

import "net/mail"

// IsValidEmail reports whether email has the syntax accepted by account
// sign-in and email confirmation.
//
// [Ja] IsValidEmail は、メールアドレスがアカウントのサインインとメール確認で
// 受理される形式かを返す。
func IsValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)

	return err == nil
}

// CanonicalEmail returns the address part of email, and whether email is
// written as that address and nothing else. It returns an empty address when
// email cannot be parsed at all.
//
// mail.ParseAddress normalises what it accepts: surrounding whitespace and a
// display name parse without error, and the address that comes back differs
// from what was written. A caller that stores the string it was handed rather
// than the address it parsed has to know the difference, because the stored
// string is what a later lookup has to match.
//
// [Ja] CanonicalEmail はメールアドレスのアドレス部分と、書かれた文字列がその
// アドレスだけであるかどうかを返す。まったく解釈できない場合は空のアドレスを返す。
//
// mail.ParseAddress は受理する値を正規化する。前後の空白や表示名はエラー無く
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

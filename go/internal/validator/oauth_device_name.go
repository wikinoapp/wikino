package validator

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// oauthDeviceNameMaxLengthは端末名の最大文字数
const oauthDeviceNameMaxLength = 100

// ParseOAuthDeviceNameは、認可要求とデバイス認可の要求の独自のパラメーター `device_name` を検証し、
// 許可に保存する端末名を返す。前後の空白を除いて空なら、送られなかったものとしてnilを返す。
// 不正UTF-8・100文字を超える値・制御文字を含む値は受け付けず、okに偽を返す (呼び出し側は `invalid_request` にする)。
//
// 端末名は連携の一覧で本人が端末を見分けるためのもので、クライアントが申告した値をそのまま信じる。
// 制御文字を拒否する理由は個人用アクセストークンの名前と同じで、一覧で読めない名前にしないためである。
func ParseOAuthDeviceName(raw string) (name *string, ok bool) {
	if !utf8.ValidString(raw) {
		return nil, false
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, true
	}
	if utf8.RuneCountInString(trimmed) > oauthDeviceNameMaxLength || strings.ContainsFunc(trimmed, unicode.IsControl) {
		return nil, false
	}
	return &trimmed, true
}

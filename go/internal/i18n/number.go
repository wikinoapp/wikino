package i18n

import (
	"context"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// FormatNumberは整数を、コンテキストの言語に合わせて桁区切りした文字列にする (例: 500000 → "500,000")。
// 桁の多い数値を文面に入れるときに、読み手が桁数を数えずに大きさを読み取れるようにするために使う
func FormatNumber(ctx context.Context, n int) string {
	return message.NewPrinter(language.Make(GetLocale(ctx))).Sprintf("%d", n)
}

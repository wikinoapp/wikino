package attachment

import (
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// maxFilenameBytesは、保存するファイル名のバイト数の上限。多くのファイルシステムの上限 (255バイト) に合わせる。
const maxFilenameBytes = 255

// maxExtensionBytesは、ファイル名を切り詰めるときに残す拡張子のバイト数の上限。
// これより長い「拡張子」は拡張子とみなさず、ほかの部分と一緒に切り詰める。
const maxExtensionBytes = 16

// replacedRunesは、パスの区切りか、Windowsでファイル名に使えないため `_` に置き換える文字。
const replacedRunes = `/\:*?"<>|`

// windowsReservedRegexpは、Windowsで装置を指すため、ファイル名の拡張子より前の部分に使えない名前。
var windowsReservedRegexp = regexp.MustCompile(`(?i)^(CON|PRN|AUX|NUL|COM[0-9¹²³]|LPT[0-9¹²³])$`)

// sanitizeFilenameは、添付ファイルの元のファイル名 (信頼できない入力) を、保存先のディレクトリの中の
// 1つのファイルを指す名前に整える。整えた結果が使えない名前になるときは、fallback (添付ファイルのID) を返す。
//
//   - 制御文字と書式文字 (文字の向きを変えて拡張子を偽装できる) を取り除く
//   - パスの区切り (`/`・`\`) とWindowsで使えない文字を `_` に置き換え、階層を移れないようにする
//   - 先頭と末尾の `.` と空白を取り除く。`..` や隠しファイル (`.bashrc` など) にならず、Windowsで末尾が落ちる名前も避ける
//   - Windowsの装置名 (`CON` など) は先頭に `_` を付ける
//   - 255バイトを超える名前は、拡張子を残して切り詰める
func sanitizeFilename(name, fallback string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r == utf8.RuneError, unicode.IsControl(r), unicode.Is(unicode.Cf, r):
			continue
		case strings.ContainsRune(replacedRunes, r):
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
	}

	s := strings.Trim(b.String(), ". ")
	s = truncateFilename(s)

	stem, _, _ := strings.Cut(s, ".")
	if windowsReservedRegexp.MatchString(strings.TrimRight(stem, " ")) {
		s = truncateFilename("_" + s)
	}

	if s == "" || !filepath.IsLocal(s) || filepath.Base(s) != s {
		return fallback
	}

	return s
}

// truncateFilenameは、sが上限のバイト数を超えるとき、拡張子を残して文字の境界で切り詰める。
func truncateFilename(s string) string {
	if len(s) <= maxFilenameBytes {
		return s
	}

	ext := filepath.Ext(s)
	if len(ext) > maxExtensionBytes {
		ext = ""
	}

	stem := s[:len(s)-len(ext)]
	limit := maxFilenameBytes - len(ext)
	for limit > 0 && !utf8.RuneStart(stem[limit]) {
		limit--
	}

	return strings.TrimRight(stem[:limit], ". ") + ext
}

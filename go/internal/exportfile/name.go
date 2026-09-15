// Package exportfileは、ページタイトルとトピック名をエクスポートのアーカイブが
// ディレクトリ名・ファイル名として使う形へ変換する。タイトルにはファイルシステムが受け付けない
// 文字や、ObsidianがWikiリンクの中で構文として読む文字が入るため、アーカイブのエントリ名は
// 入力をそのまま使わず必ずここで組み立てる。
//
// あわせて、ページ本文のリンクをそれらの名前へ書き換える。展開したディレクトリの中で、
// リンクが隣にあるファイルへ解決されるようにするためである。
package exportfile

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	// maxNameBytesはパス1要素が使えるUTF-8のバイト数で、拡張子を含む。ext4はこれを
	// 超える名前を受け付けず、日本語200文字のタイトルは600バイトに達する。
	maxNameBytes = 255

	// substituteNameは使える文字が残らなかったタイトルの代わりに使う。アーカイブに ""
	// "." ".." という名前のエントリを作らないためで、これらはそれ自体がファイルを指す名前ではない。
	substituteName = "untitled"

	// reservedDeviceNameSuffixは、そのままではWindowsのデバイスを開いてしまう名前を
	// デバイス名から引き離す。
	reservedDeviceNameSuffix = "_"

	// percentReplacementは2つ以上連続するパーセント記号の置き換え先。ObsidianはWiki
	// リンクの中でこれをコメントの開始・終了として読む。単独のパーセント記号にその意味は無いので
	// そのまま残す。
	percentReplacement = '％'
)

// fullwidthReplacementsは、名前に残せない文字を、タイトルの読みやすさを保つ全角形へ
// 対応付ける。
//
// 1組目は主要なOSがファイル名として受け付けない文字。バリデーターが弾くため "/" "\" ":" を
// 含むタイトルはここへ来ないが、表には持たせている。パス区切りの入ったエントリ名は、展開先の
// ディレクトリの外を指してしまうためである。
//
// 2組目はObsidianが [[...]] の中で構文として読む文字。"#" は見出しへのアンカー、"^" は
// ブロック参照を開き、"[" "]" は括弧の対応を壊す。4つともファイル名としては有効だが、これらを
// 含むファイルへのリンクは解決されない。
var fullwidthReplacements = map[rune]rune{
	'/':  '／',
	'\\': '＼',
	':':  '：',
	'*':  '＊',
	'?':  '？',
	'"':  '”',
	'<':  '＜',
	'>':  '＞',
	'|':  '｜',

	'#': '＃',
	'^': '＾',
	'[': '［',
	']': '］',
}

// windowsReservedDeviceNamesは、Windowsがファイルではなくデバイスとして解決する名前。
// "con.md" も "con" と同じくコンソールを開くため、比較は大文字小文字を無視し、最初のドットより
// 前の部分だけを見る。WindowsはISO-8859-1の上付き数字1・2・3もCOM・LPTデバイス名の
// 数字として扱う。
var windowsReservedDeviceNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true,
	"COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"COM¹": true, "COM²": true, "COM³": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true,
	"LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
	"LPT¹": true, "LPT²": true, "LPT³": true,
}

// Nameはアーカイブがページ・トピックに与える名前を返す。titleを変換し、末尾にextを
// 付ける。extはfilepath.Extと同じく先頭のドットを含み、ディレクトリの場合は空文字列を渡す。
// 拡張子も変換し、非常に長い場合でもベース名の場所を確保する。
//
// 返す名前はパスの1要素になる。タイトルが何であってもパス区切りを含まず "." ".." にもならない
// ため、この名前を付けたエントリは展開先のディレクトリの中に留まる。同じディレクトリの中で
// 衝突する名前はDeduperが区別する。
func Name(title, ext string) string {
	ext = safeExtension(ext, 0)
	return sanitize(title, len(ext)) + ext
}

// sanitizeはtitleを、拡張子より前の部分へ変換する。reservedは呼び出し元がその後ろに
// 足すバイト数で、名前のバイト数の上限はその分を空けておく必要がある。
func sanitize(title string, reserved int) string {
	name := orSubstitute(trimEnds(replaceRunes(title)))
	name = escapeReservedDeviceName(name)

	// 切り詰めると末尾に空白やドットが現れることがある。Windowsは渡されたファイル名から
	// それらを落とす。その文字を取り除くことで予約デバイス名が現れることもあるため、最後の名前を
	// もう一度確認する。
	name = orSubstitute(trimEnds(truncateBytes(name, maxNameBytes-reserved)))

	return escapeReservedDeviceName(name)
}

// replaceRunesは名前が持てない文字を取り除き、そのままでは残せない文字を置き換える。
// 制御文字とあわせて書字方向の制御文字も取り除く。これらは名前の残りの部分を描画する向きを
// 反転させるため、含んだままにすると、一覧に出る名前が実際とは別の拡張子で終わるファイルが
// できる。
func replaceRunes(title string) string {
	runes := make([]rune, 0, len(title))
	for _, r := range title {
		if unicode.IsControl(r) || unicode.Is(unicode.Bidi_Control, r) {
			continue
		}

		runes = append(runes, r)
	}

	var b strings.Builder
	b.Grow(len(title))

	for i := 0; i < len(runes); {
		r := runes[i]

		if r == '%' {
			run := 1
			for i+run < len(runes) && runes[i+run] == '%' {
				run++
			}

			replacement := r
			if run > 1 {
				replacement = percentReplacement
			}

			b.WriteString(strings.Repeat(string(replacement), run))
			i += run

			continue
		}

		if replacement, ok := fullwidthReplacements[r]; ok {
			r = replacement
		}

		b.WriteRune(r)
		i++
	}

	return b.String()
}

// trimEndsはnameの両端の空白とドットを取り除く。これらを残した名前は、あるシステムでは
// 隠しファイルになり、別のシステムでは受け付けられず、目に見えない差にもなる。
func trimEnds(name string) string {
	return strings.TrimFunc(name, func(r rune) bool {
		return r == '.' || unicode.IsSpace(r)
	})
}

// orSubstituteは、それ自体がファイルを指さない名前を置き換える。
func orSubstitute(name string) string {
	if name == "" || name == "." || name == ".." {
		return substituteName
	}

	return name
}

// escapeReservedDeviceNameは、Windowsがデバイスとして解決する部分 (最初のドットより前) の
// 後ろに接尾辞を付ける。
func escapeReservedDeviceName(name string) string {
	head, rest, found := strings.Cut(name, ".")

	// Windowsは最初のドットより前の部分から末尾の空白を落としてからデバイスかどうかを
	// 判定するため、"CON .txt" も "CON.txt" と同じくコンソールを開く。
	if !windowsReservedDeviceNames[strings.ToUpper(strings.TrimRight(head, " "))] {
		return name
	}

	if !found {
		return head + reservedDeviceNameSuffix
	}

	return head + reservedDeviceNameSuffix + "." + rest
}

// truncateBytesはnameをlimitバイトまで切り詰める。文字は途中で切らない。文字の途中で
// 切ると壊れたバイト列が残り、アーカイブを展開した先で文字化けした名前になる。
func truncateBytes(name string, limit int) string {
	if limit <= 0 {
		return ""
	}

	if len(name) <= limit {
		return name
	}

	cut := limit
	for cut > 0 && !utf8.RuneStart(name[cut]) {
		cut--
	}

	return name[:cut]
}

// safeExtensionは拡張子を安全化し、代替名と連番の場所を残す。
// 入力由来の添付拡張子にはタイトルと同じ危険な文字が含まれうる。
func safeExtension(ext string, suffixBytes int) string {
	ext = trimEnds(replaceRunes(ext))
	ext = trimEnds(truncateBytes(ext, maxNameBytes-len(substituteName)-suffixBytes-1))
	if ext == "" {
		return ""
	}
	return "." + ext
}

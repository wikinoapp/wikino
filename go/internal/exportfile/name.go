// Package exportfile converts page titles and topic names into the names an export archive gives
// its directories and files. A title can hold characters that a file system rejects, or that
// Obsidian reads as syntax inside a wiki link, so every entry name of the archive is built here
// rather than taken from the input as it is.
//
// The package also rewrites the links of a page body onto those names, so that a link of the
// extracted directory resolves to the file that sits beside it.
//
// [Ja] exportfile パッケージは、ページタイトルとトピック名をエクスポートのアーカイブが
// ディレクトリ名・ファイル名として使う形へ変換する。タイトルにはファイルシステムが受け付けない
// 文字や、Obsidian が Wiki リンクの中で構文として読む文字が入るため、アーカイブのエントリ名は
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
	// maxNameBytes is how many UTF-8 bytes one path component may take, extension included. ext4
	// rejects a longer name, while a title of 200 Japanese characters reaches 600 bytes.
	//
	// [Ja] maxNameBytes はパス 1 要素が使える UTF-8 のバイト数で、拡張子を含む。ext4 はこれを
	// 超える名前を受け付けず、日本語 200 文字のタイトルは 600 バイトに達する。
	maxNameBytes = 255

	// substituteName stands in for a title that is left with nothing usable. It keeps the archive
	// from holding an entry named "", "." or "..", none of which names a file of its own.
	//
	// [Ja] substituteName は使える文字が残らなかったタイトルの代わりに使う。アーカイブに ""
	// "." ".." という名前のエントリを作らないためで、これらはそれ自体がファイルを指す名前ではない。
	substituteName = "untitled"

	// reservedDeviceNameSuffix separates a name from the Windows device it would otherwise open.
	//
	// [Ja] reservedDeviceNameSuffix は、そのままでは Windows のデバイスを開いてしまう名前を
	// デバイス名から引き離す。
	reservedDeviceNameSuffix = "_"

	// percentReplacement takes the place of a run of two or more percent signs, which Obsidian
	// reads as the start and end of a comment inside a wiki link. A single percent sign carries no
	// such meaning and is left alone.
	//
	// [Ja] percentReplacement は 2 つ以上連続するパーセント記号の置き換え先。Obsidian は Wiki
	// リンクの中でこれをコメントの開始・終了として読む。単独のパーセント記号にその意味は無いので
	// そのまま残す。
	percentReplacement = '％'
)

// fullwidthReplacements maps each character that cannot stay in a name to the fullwidth form that
// keeps the title readable.
//
// The first group is what the major operating systems reject in a file name. A title never reaches
// here holding "/", "\" or ":" because the validator rejects those, yet the table still carries
// them: an entry name that grew a path separator would point outside the directory it is extracted
// into.
//
// The second group is what Obsidian reads as syntax inside [[...]]: "#" opens a heading anchor,
// "^" a block reference, and "[" "]" break the pairing of the brackets. All four are valid in a
// file name, but a link to a file holding one of them does not resolve.
//
// [Ja] fullwidthReplacements は、名前に残せない文字を、タイトルの読みやすさを保つ全角形へ
// 対応付ける。
//
// 1 組目は主要な OS がファイル名として受け付けない文字。バリデーターが弾くため "/" "\" ":" を
// 含むタイトルはここへ来ないが、表には持たせている。パス区切りの入ったエントリ名は、展開先の
// ディレクトリの外を指してしまうためである。
//
// 2 組目は Obsidian が [[...]] の中で構文として読む文字。"#" は見出しへのアンカー、"^" は
// ブロック参照を開き、"[" "]" は括弧の対応を壊す。4 つともファイル名としては有効だが、これらを
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

// windowsReservedDeviceNames are the names Windows resolves to a device instead of a file. The
// comparison ignores case and looks only at the part before the first dot, since "con.md" opens
// the console just as "con" does. Windows also treats the ISO-8859-1 superscript forms of 1, 2 and
// 3 as digits in COM and LPT device names.
//
// [Ja] windowsReservedDeviceNames は、Windows がファイルではなくデバイスとして解決する名前。
// "con.md" も "con" と同じくコンソールを開くため、比較は大文字小文字を無視し、最初のドットより
// 前の部分だけを見る。Windows は ISO-8859-1 の上付き数字 1・2・3 も COM・LPT デバイス名の
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

// Name returns the name the archive gives a page or a topic, built from title with ext appended.
// ext carries its leading dot, as filepath.Ext returns it, and is empty for a directory.
// It is sanitized too, with room reserved for the basename even if the extension is very long.
//
// The result is a single path component: it holds no path separator and is never "." or "..",
// whatever the title is, so an entry named with it stays inside the directory it is extracted
// into. Names that collide inside one directory are told apart by Deduper.
//
// [Ja] Name はアーカイブがページ・トピックに与える名前を返す。title を変換し、末尾に ext を
// 付ける。ext は filepath.Ext と同じく先頭のドットを含み、ディレクトリの場合は空文字列を渡す。
// 拡張子も変換し、非常に長い場合でもベース名の場所を確保する。
//
// 返す名前はパスの 1 要素になる。タイトルが何であってもパス区切りを含まず "." ".." にもならない
// ため、この名前を付けたエントリは展開先のディレクトリの中に留まる。同じディレクトリの中で
// 衝突する名前は Deduper が区別する。
func Name(title, ext string) string {
	ext = safeExtension(ext, 0)
	return sanitize(title, len(ext)) + ext
}

// sanitize converts title into the part of the name before the extension. reserved is the number
// of bytes the caller adds after it, which the byte budget of the name has to leave room for.
//
// [Ja] sanitize は title を、拡張子より前の部分へ変換する。reserved は呼び出し元がその後ろに
// 足すバイト数で、名前のバイト数の上限はその分を空けておく必要がある。
func sanitize(title string, reserved int) string {
	name := orSubstitute(trimEnds(replaceRunes(title)))
	name = escapeReservedDeviceName(name)

	// Cutting the name can expose a space or a dot at its end, which Windows drops from a file
	// name it is given. Trimming those characters can in turn expose a reserved device name, so the
	// final name must be checked again.
	//
	// [Ja] 切り詰めると末尾に空白やドットが現れることがある。Windows は渡されたファイル名から
	// それらを落とす。その文字を取り除くことで予約デバイス名が現れることもあるため、最後の名前を
	// もう一度確認する。
	name = orSubstitute(trimEnds(truncateBytes(name, maxNameBytes-reserved)))

	return escapeReservedDeviceName(name)
}

// replaceRunes drops the characters a name must not carry and replaces the ones that cannot stay
// in it as they are. Alongside the control characters it drops the bidirectional controls, which
// reverse the way the rest of the name is drawn: a title carrying one of them yields a file whose
// listed name ends in an extension the file does not have.
//
// [Ja] replaceRunes は名前が持てない文字を取り除き、そのままでは残せない文字を置き換える。
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

// trimEnds removes the spaces and dots at both ends of name. A name that keeps them is hidden on
// one system, refused on another, or reads as a difference the eye cannot see.
//
// [Ja] trimEnds は name の両端の空白とドットを取り除く。これらを残した名前は、あるシステムでは
// 隠しファイルになり、別のシステムでは受け付けられず、目に見えない差にもなる。
func trimEnds(name string) string {
	return strings.TrimFunc(name, func(r rune) bool {
		return r == '.' || unicode.IsSpace(r)
	})
}

// orSubstitute replaces a name that does not name a file of its own.
//
// [Ja] orSubstitute は、それ自体がファイルを指さない名前を置き換える。
func orSubstitute(name string) string {
	if name == "" || name == "." || name == ".." {
		return substituteName
	}

	return name
}

// escapeReservedDeviceName puts a suffix after the part of name that Windows resolves to a
// device, which is everything before the first dot.
//
// [Ja] escapeReservedDeviceName は、Windows がデバイスとして解決する部分 (最初のドットより前) の
// 後ろに接尾辞を付ける。
func escapeReservedDeviceName(name string) string {
	head, rest, found := strings.Cut(name, ".")

	// Windows drops the trailing spaces of the part before the first dot before it decides whether
	// the name is a device, so "CON .txt" opens the console just as "CON.txt" does.
	//
	// [Ja] Windows は最初のドットより前の部分から末尾の空白を落としてからデバイスかどうかを
	// 判定するため、"CON .txt" も "CON.txt" と同じくコンソールを開く。
	if !windowsReservedDeviceNames[strings.ToUpper(strings.TrimRight(head, " "))] {
		return name
	}

	if !found {
		return head + reservedDeviceNameSuffix
	}

	return head + reservedDeviceNameSuffix + "." + rest
}

// truncateBytes cuts name down to limit bytes, keeping whole characters. A cut through a character
// leaves a broken byte sequence, which reads as a garbled name wherever the archive is extracted.
//
// [Ja] truncateBytes は name を limit バイトまで切り詰める。文字は途中で切らない。文字の途中で
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

// safeExtension sanitizes an extension and reserves room for a fallback basename and a counter.
// Untrusted attachment extensions may contain the same unsafe characters as titles.
//
// [Ja] safeExtension は拡張子を安全化し、代替名と連番の場所を残す。
// 入力由来の添付拡張子にはタイトルと同じ危険な文字が含まれうる。
func safeExtension(ext string, suffixBytes int) string {
	ext = trimEnds(replaceRunes(ext))
	ext = trimEnds(truncateBytes(ext, maxNameBytes-len(substituteName)-suffixBytes-1))
	if ext == "" {
		return ""
	}
	return "." + ext
}

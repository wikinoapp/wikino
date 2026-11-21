package usecase

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/storage"
)

// 添付ファイルの署名付きURLで、オブジェクトストレージに返させる応答ヘッダーの決め方。
// ActiveStorageの既定と同じ規則にし、Rails期から配信してきた添付ファイルの応答を変えない。
// ActiveStorageの規則は `ActiveStorage::Blob::Servable` と `ActionDispatch::Http::ContentDisposition` にある

// attachmentBinaryContentTypesは、ブラウザが解釈して実行しうるため、表示のときに種類を
// application/octet-streamに替えて、ダウンロードさせる種類 (ActiveStorageのcontent_types_to_serve_as_binary)
var attachmentBinaryContentTypes = []string{
	"text/html",
	"image/svg+xml",
	"application/postscript",
	"application/x-shockwave-flash",
	"text/xml",
	"application/xml",
	"application/xhtml+xml",
	"application/mathml+xml",
	"text/cache-manifest",
}

// attachmentInlineContentTypesは、ブラウザの中で開かせる種類 (ActiveStorageのcontent_types_allowed_inline)。
// これ以外の種類はダウンロードさせる
var attachmentInlineContentTypes = []string{
	"image/webp",
	"image/avif",
	"image/png",
	"image/gif",
	"image/jpeg",
	"image/tiff",
	"image/bmp",
	"image/vnd.adobe.photoshop",
	"image/vnd.microsoft.icon",
	"application/pdf",
}

// attachmentBinaryContentTypeは、バイナリとして返す種類の代わりに返すContent-Type
const attachmentBinaryContentType = "application/octet-stream"

// attachmentDeliveryOptionsは、添付ファイルを表示するための署名付きURLに含める応答ヘッダーを返す。
// `/attachments/{id}` のリダイレクト先に使う。ブラウザが解釈して実行しうる種類は、種類を
// application/octet-streamに替えてダウンロードさせる
func attachmentDeliveryOptions(attachment *model.Attachment) storage.PresignedGetOptions {
	contentType := attachment.ContentType
	if isAttachmentServedAsBinary(contentType) {
		contentType = attachmentBinaryContentType
	}
	return storage.PresignedGetOptions{
		ResponseContentDisposition: attachmentContentDisposition(attachment),
		ResponseContentType:        contentType,
	}
}

// attachmentSignedURLOptionsは、本文の中で添付ファイルを表示するための署名付きURLに含める応答ヘッダーを返す。
// `POST /attachments/signed_urls` に使う。Content-Dispositionは表示と同じ規則で決め、Content-Typeは記録された
// 種類のまま返す。SVGを `<img>` で表示できるようにするためで、ブラウザは `<img>` の中のSVGのスクリプトを実行しない
func attachmentSignedURLOptions(attachment *model.Attachment) storage.PresignedGetOptions {
	return storage.PresignedGetOptions{
		ResponseContentDisposition: attachmentContentDisposition(attachment),
		ResponseContentType:        attachment.ContentType,
	}
}

// attachmentContentDispositionは、添付ファイルの応答のContent-Dispositionを返す。
// インラインで開かせる種類だけをinlineにし、それ以外とバイナリとして返す種類はattachmentにする。
// どちらにもファイル名を付ける
func attachmentContentDisposition(attachment *model.Attachment) string {
	disposition := "inline"
	if isAttachmentServedAsBinary(attachment.ContentType) || !slices.Contains(attachmentInlineContentTypes, attachment.ContentType) {
		disposition = "attachment"
	}
	return formatContentDisposition(disposition, sanitizeAttachmentFilename(attachment.Filename))
}

// isAttachmentServedAsBinaryは、種類contentTypeをバイナリとして返すかどうかを返す
func isAttachmentServedAsBinary(contentType string) bool {
	return slices.Contains(attachmentBinaryContentTypes, contentType)
}

// attachmentFilenameReplacerは、Content-Dispositionのファイル名に入れない文字を「-」に置き換える
// (ActiveStorage::Filename#sanitizedと同じ文字)
var attachmentFilenameReplacer = strings.NewReplacer(
	"\u202e", "-", "\x00", "-", "%", "-", "$", "-", "|", "-", ":", "-", ";", "-", "/", "-",
	"<", "-", ">", "-", "?", "-", "*", "-", `"`, "-", "\t", "-", "\r", "-", "\n", "-", `\`, "-",
)

// sanitizeAttachmentFilenameは、Content-Dispositionに入れるファイル名を整える。不正なUTF-8を
// U+FFFDに替え、前後の空白を除き、パスの区切りやヘッダーを壊しうる文字を「-」に置き換える
func sanitizeAttachmentFilename(filename string) string {
	filename = strings.ToValidUTF8(filename, "\uFFFD")
	filename = strings.Trim(filename, " \t\n\v\f\r\x00")
	return attachmentFilenameReplacer.Replace(filename)
}

// formatContentDispositionは、dispositionとファイル名からContent-Dispositionを組み立てる。
// ASCIIだけを読むクライアントのための `filename` と、UTF-8のファイル名を伝える `filename*` (RFC 6266・RFC 5987)
// を両方付ける
func formatContentDisposition(disposition, filename string) string {
	ascii := percentEscape(asciiApproximation(filename), isTraditionalFilenameChar)
	utf8Name := percentEscape(filename, isRFC5987FilenameChar)
	return fmt.Sprintf(`%s; filename="%s"; filename*=UTF-8''%s`, disposition, ascii, utf8Name)
}

// asciiApproximationは、ファイル名をASCIIの文字に近づける。アクセント記号などの結合文字は
// 取り除いて元の文字を残し、ASCIIに無い文字は「?」にする
func asciiApproximation(s string) string {
	var b strings.Builder
	for _, r := range norm.NFKD.String(s) {
		switch {
		case unicode.Is(unicode.Mn, r):
			continue
		case r < utf8.RuneSelf:
			b.WriteRune(r)
		default:
			b.WriteByte('?')
		}
	}
	return b.String()
}

// percentEscapeは、keepが偽を返す文字をUTF-8のバイトごとに「%XX」へ置き換える
func percentEscape(s string, keep func(rune) bool) string {
	var b strings.Builder
	for _, r := range s {
		if keep(r) {
			b.WriteRune(r)
			continue
		}
		var buf [utf8.UTFMax]byte
		n := utf8.EncodeRune(buf[:], r)
		for _, c := range buf[:n] {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

// isTraditionalFilenameCharは、`filename` の引用符の中にそのまま置く文字かどうかを返す
func isTraditionalFilenameChar(r rune) bool {
	return isASCIIAlphanumeric(r) || strings.ContainsRune(" !#$+.^_`|~-", r)
}

// isRFC5987FilenameCharは、`filename*` にそのまま置く文字 (RFC 5987のattr-char) かどうかを返す
func isRFC5987FilenameChar(r rune) bool {
	return isASCIIAlphanumeric(r) || strings.ContainsRune("!#$&+.^_`|~-", r)
}

// isASCIIAlphanumericは、ASCIIの英数字かどうかを返す
func isASCIIAlphanumeric(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

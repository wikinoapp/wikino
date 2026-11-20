package validator

import (
	"context"
	"encoding/base64"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// attachmentFilenameMaxBytesは、整えたあとのファイル名の上限のバイト数
const attachmentFilenameMaxBytes = 255

// attachmentChecksumBytesは、チェックサム (MD5) のバイト数
const attachmentChecksumBytes = 16

// attachmentFilenameTrimCharsは、ファイル名の前後から除く文字 (RubyのString#stripと同じ空白とNUL)
const attachmentFilenameTrimChars = " \t\n\v\f\r\x00"

var (
	// attachmentFilenameInvalidCharsは、ファイル名に使えない文字 (Windowsで使えない文字と制御文字) とパスの区切り
	attachmentFilenameInvalidChars = regexp.MustCompile(`[<>:"|?*\x00-\x1f/\\]`)
	// attachmentFilenameConsecutiveDotsは、パスを遡る形を作らないよう置き換える連続するドット
	attachmentFilenameConsecutiveDots = regexp.MustCompile(`\.{2,}`)
	// attachmentFilenameConsecutiveUnderscoresは、置き換えで続いたアンダースコアを1つにまとめるための並び
	attachmentFilenameConsecutiveUnderscores = regexp.MustCompile(`_{2,}`)
)

// attachmentFilenameReservedNamesは、Windowsが装置の名前として予約しているため、ファイル名の本体に使えない名前
var attachmentFilenameReservedNames = []string{
	"CON", "PRN", "AUX", "NUL",
	"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
	"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
}

// AttachmentPresignCreateValidatorは、添付ファイルのアップロードの事前署名のリクエストを検証する。
// 検証と、ファイル名の整え方は、Rails版のAttachments::PresignFormと同じ結果にする
type AttachmentPresignCreateValidator struct{}

// NewAttachmentPresignCreateValidatorはAttachmentPresignCreateValidatorを生成する
func NewAttachmentPresignCreateValidator() *AttachmentPresignCreateValidator {
	return &AttachmentPresignCreateValidator{}
}

// AttachmentPresignCreateValidatorInputは事前署名の検証の入力パラメータ
type AttachmentPresignCreateValidatorInput struct {
	Filename    string
	ContentType string
	// ByteSizeはアップロードする大きさ。リクエストに無ければnil
	ByteSize *int64
	// Checksumは内容のMD5をBase64にした値
	Checksum string
}

// Validateは事前署名のリクエストを検証し、成功したら整えたファイル名を返す
func (v *AttachmentPresignCreateValidator) Validate(ctx context.Context, input AttachmentPresignCreateValidatorInput) (string, error) {
	ve := model.NewValidationError()

	filename := sanitizeAttachmentUploadFilename(input.Filename)
	if filename == "" {
		ve.AddField("filename", i18n.T(ctx, "validation_attachment_filename_required"))
	}

	switch {
	case input.ContentType == "":
		ve.AddField("content_type", i18n.T(ctx, "validation_attachment_content_type_required"))
	case !model.IsAttachmentContentTypeAllowed(input.ContentType):
		ve.AddField("content_type", i18n.T(ctx, "validation_attachment_content_type_unsupported"))
	}

	switch {
	case input.ByteSize == nil:
		ve.AddField("byte_size", i18n.T(ctx, "validation_attachment_byte_size_required"))
	case input.ContentType == "":
		// Rails版と同じく、種類が分からないときは大きさの値を検査しない
	case *input.ByteSize <= 0:
		ve.AddField("byte_size", i18n.T(ctx, "validation_attachment_byte_size_not_positive"))
	default:
		if maxSize := model.AttachmentMaxByteSize(input.ContentType); *input.ByteSize > maxSize {
			ve.AddField("byte_size", i18n.T(ctx, "validation_attachment_byte_size_too_large", map[string]any{
				"MaxMB": maxSize / (1024 * 1024),
			}))
		}
	}

	// Rails版は空でないことだけを確かめていた。Go版はチェックサムをアップロードの署名に含めるため、
	// MD5のBase64として読めることも確かめる。読めない値で署名すると、アップロードが必ず失敗する
	switch {
	case input.Checksum == "":
		ve.AddField("checksum", i18n.T(ctx, "validation_attachment_checksum_required"))
	case !isAttachmentChecksum(input.Checksum):
		ve.AddField("checksum", i18n.T(ctx, "validation_attachment_checksum_invalid"))
	}

	if ve.HasErrors() {
		return "", ve
	}
	return filename, nil
}

// isAttachmentChecksumは、checksumがMD5 (16バイト) をパディング付きのBase64にした値かどうかを返す
func isAttachmentChecksum(checksum string) bool {
	// StrictでもCRとLFは読み飛ばされるため、HTTPヘッダーに送れない値を受け入れないよう長さも確かめる
	if len(checksum) != base64.StdEncoding.EncodedLen(attachmentChecksumBytes) {
		return false
	}
	decoded, err := base64.StdEncoding.Strict().DecodeString(checksum)
	return err == nil && len(decoded) == attachmentChecksumBytes
}

// sanitizeAttachmentUploadFilenameは、アップロードするファイル名を整える。空白だけのファイル名は空にする。
// Unicodeを正規化し、前後の空白を除き、使えない文字とパスの区切りと連続するドットを「_」に置き換え、
// 先頭のドットを除き、Windowsの予約名を避けて、拡張子を保ったまま255バイトに切り詰める
func sanitizeAttachmentUploadFilename(filename string) string {
	if strings.TrimFunc(filename, unicode.IsSpace) == "" {
		return ""
	}

	name := norm.NFC.String(filename)
	name = strings.Trim(name, attachmentFilenameTrimChars)
	name = attachmentFilenameInvalidChars.ReplaceAllString(name, "_")
	name = attachmentFilenameConsecutiveDots.ReplaceAllString(name, "_")
	name = attachmentFilenameConsecutiveUnderscores.ReplaceAllString(name, "_")
	name = strings.TrimLeft(name, ".")

	basename, extension := splitAttachmentFilenameExtension(name)
	if basename == "" {
		basename = "file"
	}
	if slices.Contains(attachmentFilenameReservedNames, strings.ToUpper(basename)) {
		basename += "_file"
	}
	name = basename + extension

	if len(name) <= attachmentFilenameMaxBytes {
		return name
	}
	// 拡張子を保てるときは本体だけを切り詰め、拡張子だけで上限を超えるときは全体を切り詰める。
	// 切り詰めで途中になったUTF-8の文字は取り除く
	if maxBasenameBytes := attachmentFilenameMaxBytes - len(extension); maxBasenameBytes > 0 {
		return strings.ToValidUTF8(basename[:maxBasenameBytes], "") + extension
	}
	return strings.ToValidUTF8(name[:attachmentFilenameMaxBytes], "")
}

// splitAttachmentFilenameExtensionは、ファイル名を本体と拡張子 (最後のドットから後ろ) に分ける。
// 先頭のドットは拡張子の始まりとみなさない (RubyのFile.extnameと同じ)
func splitAttachmentFilenameExtension(name string) (basename, extension string) {
	i := strings.LastIndex(name, ".")
	if i <= 0 {
		return name, ""
	}
	return name[:i], name[i:]
}

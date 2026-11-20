package validator_test

import (
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// validPresignChecksumは空の内容のMD5をBase64にした値
const validPresignChecksum = "1B2M2Y8AsgTpgAmY7PhCfg=="

func presignByteSize(n int64) *int64 { return &n }

// TestAttachmentPresignCreateValidator_Filenameは、整えたファイル名がRails版 (Attachments::PresignForm) と
// 同じになることを確かめる。期待値は、同じ入力をRails版のフォームに渡して得た値である
func TestAttachmentPresignCreateValidator_Filename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "そのまま使えるファイル名", input: "photo.png", want: "photo.png"},
		{name: "前後の空白を除く", input: "  photo.png  ", want: "photo.png"},
		{name: "パスを遡る形を置き換える", input: "../../etc/passwd", want: "_etc_passwd"},
		{name: "パスの区切りを置き換える", input: `a/b\c.txt`, want: "a_b_c.txt"},
		{name: "連続するドットを置き換える", input: "a..b.txt", want: "a_b.txt"},
		{name: "3つ以上のドットも1つの「_」にする", input: "a...b", want: "a_b"},
		{name: "先頭の連続するドットは「_」になる", input: "...hidden", want: "_hidden"},
		{name: "先頭のドットを除く", input: ".bashrc", want: "bashrc"},
		{name: "先頭のドットを除いたあとの拡張子", input: "..png", want: "_png"},
		{name: "使えない文字を置き換えて1つにまとめる", input: `a<>:"|?*b.txt`, want: "a_b.txt"},
		{name: "制御文字を置き換える", input: "tab\there.txt", want: "tab_here.txt"},
		{name: "NULを置き換える", input: "file\x00name.txt", want: "file_name.txt"},
		{name: "連続するアンダースコアを1つにする", input: "__a___b__.txt", want: "_a_b_.txt"},
		{name: "予約名の本体に「_file」を付ける", input: "CON.txt", want: "CON_file.txt"},
		{name: "予約名は大文字と小文字を区別しない", input: "con", want: "con_file"},
		{name: "末尾のドットの前の予約名", input: "nul.", want: "nul_file."},
		{name: "数字の付く予約名", input: "COM1.", want: "COM1_file."},
		{name: "拡張子が予約名でも本体が予約名でなければそのまま", input: "x.CON", want: "x.CON"},
		{name: "拡張子が2つあれば最後だけを拡張子とみなす", input: "con.tar.gz", want: "con.tar.gz"},
		{name: "予約名に数字を足した名前はそのまま", input: "lpt9.tar.gz", want: "lpt9.tar.gz"},
		{name: "末尾のドットは残す", input: "foo.", want: "foo."},
		{name: "ドットだけなら本体を「file」にする", input: ".", want: "file"},
		{name: "ドット2つは「_」になる", input: "..", want: "_"},
		{name: "拡張子の無いファイル名", input: "noext", want: "noext"},
		{name: "日本語のファイル名", input: "日本語のファイル.pdf", want: "日本語のファイル.pdf"},
		{name: "結合文字を正規化する", input: "é.txt", want: "é.txt"},
		{name: "全角の空白は除かない", input: "　全角空白　.txt", want: "　全角空白　.txt"},
		{name: "拡張子の前の空白は残す", input: "photo .png", want: "photo .png"},
		{name: "長いファイル名は拡張子を保って切り詰める", input: strings.Repeat("a", 300) + ".png", want: strings.Repeat("a", 251) + ".png"},
		{name: "切り詰めで途中になった文字を除く", input: strings.Repeat("あ", 100) + ".png", want: strings.Repeat("あ", 83) + ".png"},
		{name: "拡張子だけで上限を超えれば全体を切り詰める", input: "x." + strings.Repeat("e", 300), want: "x." + strings.Repeat("e", 253)},
		{name: "切り詰めたあとちょうど255バイトになる", input: "a" + strings.Repeat("あ", 90) + ".jpeg", want: "a" + strings.Repeat("あ", 83) + ".jpeg"},
	}

	v := validator.NewAttachmentPresignCreateValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(context.Background(), "ja")
			got, err := v.Validate(ctx, validator.AttachmentPresignCreateValidatorInput{
				Filename:    tt.input,
				ContentType: "image/png",
				ByteSize:    presignByteSize(1),
				Checksum:    validPresignChecksum,
			})
			if err != nil {
				t.Fatalf("Validate()のエラー = %v", err)
			}
			if got != tt.want {
				t.Errorf("ファイル名 = %q、期待値 = %q", got, tt.want)
			}
			if len(got) > 255 {
				t.Errorf("ファイル名のバイト数 = %d、期待値 = 255以下", len(got))
			}
		})
	}
}

func TestAttachmentPresignCreateValidator_Validate(t *testing.T) {
	t.Parallel()

	valid := validator.AttachmentPresignCreateValidatorInput{
		Filename:    "photo.png",
		ContentType: "image/png",
		ByteSize:    presignByteSize(1024),
		Checksum:    validPresignChecksum,
	}

	tests := []struct {
		name      string
		modify    func(in *validator.AttachmentPresignCreateValidatorInput)
		wantField string
		wantMsg   string
	}{
		{name: "正常系: 画像の上限ちょうど", modify: func(in *validator.AttachmentPresignCreateValidatorInput) {
			in.ByteSize = presignByteSize(10 * 1024 * 1024)
		}},
		{name: "正常系: 動画は100MBまで", modify: func(in *validator.AttachmentPresignCreateValidatorInput) {
			in.ContentType = "video/mp4"
			in.ByteSize = presignByteSize(100 * 1024 * 1024)
		}},
		{name: "正常系: PDF", modify: func(in *validator.AttachmentPresignCreateValidatorInput) { in.ContentType = "application/pdf" }},
		{name: "異常系: ファイル名が空", modify: func(in *validator.AttachmentPresignCreateValidatorInput) { in.Filename = "" },
			wantField: "filename", wantMsg: "ファイル名を入力してください"},
		{name: "異常系: ファイル名が空白だけ", modify: func(in *validator.AttachmentPresignCreateValidatorInput) { in.Filename = " 　 " },
			wantField: "filename", wantMsg: "ファイル名を入力してください"},
		{name: "異常系: ファイル形式が空", modify: func(in *validator.AttachmentPresignCreateValidatorInput) { in.ContentType = "" },
			wantField: "content_type", wantMsg: "ファイル形式を入力してください"},
		{name: "異常系: 許可しないファイル形式", modify: func(in *validator.AttachmentPresignCreateValidatorInput) { in.ContentType = "application/x-msdownload" },
			wantField: "content_type", wantMsg: "ファイル形式はサポートされていない形式です"},
		{name: "異常系: HTMLは許可しない", modify: func(in *validator.AttachmentPresignCreateValidatorInput) { in.ContentType = "text/html" },
			wantField: "content_type", wantMsg: "ファイル形式はサポートされていない形式です"},
		{name: "異常系: 大きさが無い", modify: func(in *validator.AttachmentPresignCreateValidatorInput) { in.ByteSize = nil },
			wantField: "byte_size", wantMsg: "ファイルサイズを入力してください"},
		{name: "異常系: 大きさが0", modify: func(in *validator.AttachmentPresignCreateValidatorInput) { in.ByteSize = presignByteSize(0) },
			wantField: "byte_size", wantMsg: "ファイルサイズは0より大きい値にしてください"},
		{name: "異常系: 大きさが負", modify: func(in *validator.AttachmentPresignCreateValidatorInput) { in.ByteSize = presignByteSize(-1) },
			wantField: "byte_size", wantMsg: "ファイルサイズは0より大きい値にしてください"},
		{name: "異常系: 画像の上限を超える", modify: func(in *validator.AttachmentPresignCreateValidatorInput) {
			in.ByteSize = presignByteSize(10*1024*1024 + 1)
		},
			wantField: "byte_size", wantMsg: "ファイルサイズは10MB以下にしてください"},
		{name: "異常系: 動画の上限を超える", modify: func(in *validator.AttachmentPresignCreateValidatorInput) {
			in.ContentType = "video/mp4"
			in.ByteSize = presignByteSize(100*1024*1024 + 1)
		}, wantField: "byte_size", wantMsg: "ファイルサイズは100MB以下にしてください"},
		{name: "異常系: そのほかの種類の上限を超える", modify: func(in *validator.AttachmentPresignCreateValidatorInput) {
			in.ContentType = "application/pdf"
			in.ByteSize = presignByteSize(25*1024*1024 + 1)
		}, wantField: "byte_size", wantMsg: "ファイルサイズは25MB以下にしてください"},
		{name: "異常系: チェックサムが無い", modify: func(in *validator.AttachmentPresignCreateValidatorInput) { in.Checksum = "" },
			wantField: "checksum", wantMsg: "チェックサムを入力してください"},
		{name: "異常系: チェックサムがBase64でない", modify: func(in *validator.AttachmentPresignCreateValidatorInput) { in.Checksum = "not base64!" },
			wantField: "checksum", wantMsg: "チェックサムの形式が正しくありません"},
		{name: "異常系: チェックサムが16バイトでない", modify: func(in *validator.AttachmentPresignCreateValidatorInput) { in.Checksum = "YWJj" },
			wantField: "checksum", wantMsg: "チェックサムの形式が正しくありません"},
		{name: "異常系: チェックサムが16進数", modify: func(in *validator.AttachmentPresignCreateValidatorInput) {
			in.Checksum = "d41d8cd98f00b204e9800998ecf8427e"
		},
			wantField: "checksum", wantMsg: "チェックサムの形式が正しくありません"},
	}

	v := validator.NewAttachmentPresignCreateValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := valid
			tt.modify(&input)
			ctx := i18n.SetLocale(context.Background(), "ja")
			_, err := v.Validate(ctx, input)

			if tt.wantField == "" {
				if err != nil {
					t.Fatalf("Validate()のエラー = %v", err)
				}
				return
			}
			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatalf("エラー = %v、ValidationErrorを期待", err)
			}
			errs := ve.GetFieldErrors(tt.wantField)
			if len(errs) != 1 || errs[0] != tt.wantMsg {
				t.Errorf("%qのエラー = %q、期待値 = [%q]", tt.wantField, errs, tt.wantMsg)
			}
		})
	}

	t.Run("英語の上限のメッセージ", func(t *testing.T) {
		t.Parallel()

		input := valid
		input.ByteSize = presignByteSize(10*1024*1024 + 1)
		_, err := v.Validate(i18n.SetLocale(context.Background(), "en"), input)
		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatalf("エラー = %v、ValidationErrorを期待", err)
		}
		if got := ve.GetFieldErrors("byte_size"); len(got) != 1 || got[0] != "File size must be 10 MB or less" {
			t.Errorf("byte_sizeのエラー = %q、期待値 = %q", got, "File size must be 10 MB or less")
		}
	})

	// Rails版と同じく、種類が分からないときは大きさの値を検査せず、ファイル形式のエラーだけを返す
	for _, tc := range []struct {
		name     string
		byteSize int64
	}{
		{name: "異常系: ファイル形式が空なら大きさが0でも大きさのエラーを返さない", byteSize: 0},
		{name: "異常系: ファイル形式が空なら大きさが負でも大きさのエラーを返さない", byteSize: -1},
		{name: "異常系: ファイル形式が空なら大きさが上限を超えても大きさのエラーを返さない", byteSize: 100*1024*1024 + 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			input := valid
			input.ContentType = ""
			input.ByteSize = presignByteSize(tc.byteSize)
			_, err := v.Validate(i18n.SetLocale(context.Background(), "ja"), input)
			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatalf("エラー = %v、ValidationErrorを期待", err)
			}
			if got := ve.GetFieldErrors("content_type"); len(got) != 1 || got[0] != "ファイル形式を入力してください" {
				t.Errorf("content_typeのエラー = %q、期待値 = [ファイル形式を入力してください]", got)
			}
			if got := ve.GetFieldErrors("byte_size"); len(got) != 0 {
				t.Errorf("byte_sizeのエラー = %q、期待値 = なし", got)
			}
		})
	}
}

// TestAttachmentPresignCreateValidator_ChecksumNewlinesは、HTTPヘッダーに送れない改行入りのチェックサムを拒否することを確かめる
func TestAttachmentPresignCreateValidator_ChecksumNewlines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		checksum string
	}{
		{name: "先頭のLF", checksum: "\n" + validPresignChecksum},
		{name: "途中のLF", checksum: validPresignChecksum[:4] + "\n" + validPresignChecksum[4:]},
		{name: "末尾のLF", checksum: validPresignChecksum + "\n"},
		{name: "先頭のCR", checksum: "\r" + validPresignChecksum},
		{name: "途中のCR", checksum: validPresignChecksum[:4] + "\r" + validPresignChecksum[4:]},
		{name: "末尾のCR", checksum: validPresignChecksum + "\r"},
		{name: "先頭のCRLF", checksum: "\r\n" + validPresignChecksum},
		{name: "途中のCRLF", checksum: validPresignChecksum[:4] + "\r\n" + validPresignChecksum[4:]},
		{name: "末尾のCRLF", checksum: validPresignChecksum + "\r\n"},
	}

	v := validator.NewAttachmentPresignCreateValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := v.Validate(i18n.SetLocale(context.Background(), "ja"), validator.AttachmentPresignCreateValidatorInput{
				Filename:    "photo.png",
				ContentType: "image/png",
				ByteSize:    presignByteSize(1234),
				Checksum:    tt.checksum,
			})
			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatalf("エラー = %v、期待値 = checksumのValidationError", err)
			}
			if got := ve.GetFieldErrors("checksum"); len(got) != 1 || got[0] != "チェックサムの形式が正しくありません" {
				t.Errorf("checksumのエラー = %q、期待値 = [チェックサムの形式が正しくありません]", got)
			}
		})
	}
}

package attachment

import (
	"strings"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	t.Parallel()

	const fallback = "01234567-89ab-cdef-0123-456789abcdef"

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "そのまま使える名前", in: "diagram.png", want: "diagram.png"},
		{name: "日本語と空白", in: "設計 図.png", want: "設計 図.png"},
		{name: "親の階層を指す", in: "../../.bashrc", want: "_.._.bashrc"},
		{name: "絶対パス", in: "/etc/passwd", want: "_etc_passwd"},
		{name: "Windowsのパスの区切りとドライブ", in: `C:\Users\evil.exe`, want: "C__Users_evil.exe"},
		{name: "..だけ", in: "..", want: fallback},
		{name: "隠しファイル", in: ".envrc", want: "envrc"},
		{name: "末尾の.と空白", in: "report.pdf. . ", want: "report.pdf"},
		{name: "制御文字", in: "a\nb\x00c\x1b[31m.png", want: "abc[31m.png"},
		{name: "文字の向きを変える書式文字", in: "invoice\u202egnp.exe", want: "invoicegnp.exe"},
		{name: "不正なUTF-8", in: "a\xffb.png", want: "ab.png"},
		{name: "Windowsの装置名", in: "con.txt", want: "_con.txt"},
		{name: "Windowsの装置名と同じ始まりの名前", in: "console.txt", want: "console.txt"},
		{name: "Windowsで使えない文字", in: `a:b*c?d"e<f>g|h.png`, want: "a_b_c_d_e_f_g_h.png"},
		{name: "空", in: "", want: fallback},
		{name: "取り除くと空になる", in: "\n\t. ", want: fallback},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := sanitizeFilename(tt.in, fallback); got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q、期待値 = %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSanitizeFilename_Truncate(t *testing.T) {
	t.Parallel()

	got := sanitizeFilename(strings.Repeat("あ", 100)+".png", "fallback")

	if len(got) > maxFilenameBytes {
		t.Errorf("%dバイトに切り詰められていない: %dバイト", maxFilenameBytes, len(got))
	}
	if !strings.HasSuffix(got, "あ.png") {
		t.Errorf("文字の境界で切り詰め、拡張子を残すことを期待したが %q だった", got)
	}
}

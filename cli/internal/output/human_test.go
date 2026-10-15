package output

import (
	"testing"
)

func TestPrinterPrintTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		color   bool
		environ []string
		headers []string
		rows    [][]string
		want    string
	}{
		{
			name:    "全角文字を幅2として列を揃える",
			headers: []string{"番号", "タイトル", "URL"},
			rows: [][]string{
				{"1", "議事録", "https://example.com/1"},
				{"12", "Meeting notes", "https://example.com/12"},
			},
			want: "番号  タイトル       URL\n" +
				"1     議事録         https://example.com/1\n" +
				"12    Meeting notes  https://example.com/12\n",
		},
		{
			name:    "セルの改行・タブ・エスケープシーケンスを空白に置き換える",
			headers: []string{"番号", "タイトル"},
			rows:    [][]string{{"1", "1行目\n2行目\t\x1b[31m赤"}},
			want: "番号  タイトル\n" +
				"1     1行目 2行目  [31m赤\n",
		},
		{
			name:    "双方向テキストの制御文字を空白に置き換える",
			headers: []string{"番号", "タイトル"},
			rows:    [][]string{{"1", "公開\u202eしない\u202c・\u2066分離\u2069"}},
			want: "番号  タイトル\n" +
				"1     公開 しない ・ 分離 \n",
		},
		{
			name:    "TTYでは見出しを太字にする",
			color:   true,
			headers: []string{"番号", "タイトル"},
			rows:    [][]string{{"1", "議事録"}},
			want: "\x1b[1m番号  タイトル\x1b[0m\n" +
				"1     議事録\n",
		},
		{
			name:    "NO_COLORなら太字にしない",
			color:   true,
			environ: []string{"NO_COLOR=1"},
			headers: []string{"番号", "タイトル"},
			rows:    [][]string{{"1", "議事録"}},
			want: "番号  タイトル\n" +
				"1     議事録\n",
		},
		{
			name:    "0件なら見出しだけを出す",
			headers: []string{"番号", "タイトル"},
			want:    "番号  タイトル\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p, stdout, err := newTestPrinter(t, printerSetup{stdoutTTY: tt.color, environ: tt.environ})
			if err != nil {
				t.Fatal(err)
			}
			if err := p.PrintTable(tt.headers, tt.rows); err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}
			if got := stdout.String(); got != tt.want {
				t.Errorf("標準出力 =\n%q\n期待値 =\n%q", got, tt.want)
			}
		})
	}
}

func TestPrinterPrintDetail(t *testing.T) {
	t.Parallel()

	fields := []Field{
		{Label: "番号", Value: "12"},
		{Label: "タイトル", Value: "議事録\x1b]0;偽のタイトル\x07"},
		{Label: "URL", Value: "https://example.com/12"},
	}

	tests := []struct {
		name  string
		color bool
		body  string
		want  string
	}{
		{
			name: "ラベルの幅を揃え、本文をそのまま続ける",
			body: "# 見出し\n\n- 項目\n\tタブ\n",
			want: "番号:     12\n" +
				"タイトル: 議事録 ]0;偽のタイトル \n" +
				"URL:      https://example.com/12\n" +
				"\n# 見出し\n\n- 項目\n\tタブ\n",
		},
		{
			name: "本文の末尾に改行が無ければ足す",
			body: "本文",
			want: "番号:     12\n" +
				"タイトル: 議事録 ]0;偽のタイトル \n" +
				"URL:      https://example.com/12\n" +
				"\n本文\n",
		},
		{
			name: "本文の改行以外の制御文字を取り除き、CRLFをLFにする",
			body: "1行目\r\n\x1b[2J2行目\r上書き\n",
			want: "番号:     12\n" +
				"タイトル: 議事録 ]0;偽のタイトル \n" +
				"URL:      https://example.com/12\n" +
				"\n1行目\n[2J2行目上書き\n",
		},
		{
			name: "本文の双方向テキストの制御文字は他の言語の文章のために残す",
			body: "\u2067שלום\u2069\n",
			want: "番号:     12\n" +
				"タイトル: 議事録 ]0;偽のタイトル \n" +
				"URL:      https://example.com/12\n" +
				"\n\u2067שלום\u2069\n",
		},
		{
			name: "本文が空なら項目だけを出す",
			body: "",
			want: "番号:     12\n" +
				"タイトル: 議事録 ]0;偽のタイトル \n" +
				"URL:      https://example.com/12\n",
		},
		{
			name:  "TTYではラベルを太字にする",
			color: true,
			body:  "",
			want: "\x1b[1m番号:\x1b[0m     12\n" +
				"\x1b[1mタイトル:\x1b[0m 議事録 ]0;偽のタイトル \n" +
				"\x1b[1mURL:\x1b[0m      https://example.com/12\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p, stdout, err := newTestPrinter(t, printerSetup{stdoutTTY: tt.color})
			if err != nil {
				t.Fatal(err)
			}
			if err := p.PrintDetail(fields, tt.body); err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}
			if got := stdout.String(); got != tt.want {
				t.Errorf("標準出力 =\n%q\n期待値 =\n%q", got, tt.want)
			}
		})
	}
}

func TestDisplayWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		s    string
		want int
	}{
		{s: "abc", want: 3},
		{s: "議事録", want: 6},
		{s: "ｶﾅ", want: 2},
		{s: "Ａ", want: 2},
		{s: "é", want: 1},
		{s: "", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			t.Parallel()

			if got := displayWidth(tt.s); got != tt.want {
				t.Errorf("displayWidth(%q) = %d、期待値 = %d", tt.s, got, tt.want)
			}
		})
	}
}

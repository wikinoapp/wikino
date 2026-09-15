package markup

import (
	"strconv"
	"strings"
	"testing"
)

func BenchmarkScanWikilinkMatchesWithManyRawHTMLRanges(b *testing.B) {
	for _, count := range []int{256, 512, 1024, 2048} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			body := strings.Repeat("<code>[[hidden]]</code> [[visible]]\n\n", count)

			b.ReportAllocs()
			b.SetBytes(int64(len(body)))
			b.ResetTimer()
			for range b.N {
				ScanWikilinkMatches(body, "topic")
			}
		})
	}
}

func BenchmarkScanAttachmentRefMatchesWithManyRawHTMLRanges(b *testing.B) {
	for _, count := range []int{256, 512, 1024, 2048} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			body := strings.Repeat(
				"<code>![hidden](/attachments/01ABC)</code> "+
					"![visible](/attachments/01XYZ)\n\n",
				count,
			)

			b.ReportAllocs()
			b.SetBytes(int64(len(body)))
			b.ResetTimer()
			for range b.N {
				ScanAttachmentRefMatches(body)
			}
		})
	}
}

func BenchmarkScanWikilinkMatchesWithParsedLabels(b *testing.B) {
	for _, count := range []int{256, 512, 1024, 2048} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			body := strings.Repeat(
				"[<span title=\"]\">label</span> ![image](https://example.com/image]x.png) [[hidden]]](https://example.com/) [[visible]]\n\n",
				count,
			)

			b.ReportAllocs()
			b.SetBytes(int64(len(body)))
			b.ResetTimer()
			for range b.N {
				ScanWikilinkMatches(body, "topic")
			}
		})
	}
}

func BenchmarkScanAttachmentRefMatchesWithParsedLabels(b *testing.B) {
	for _, count := range []int{256, 512, 1024, 2048} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			body := strings.Repeat(
				"> [<span title=\"]\">label</span> ![image](/attachments/01ABC \"[\")](\n>   /attachments/01XYZ)\n\n",
				count,
			)

			b.ReportAllocs()
			b.SetBytes(int64(len(body)))
			b.ResetTimer()
			for range b.N {
				ScanAttachmentRefMatches(body)
			}
		})
	}
}

func BenchmarkScanWikilinkMatchesWithNestedElements(b *testing.B) {
	for _, shape := range []string{"matched", "mismatched", "interior"} {
		b.Run(shape, func(b *testing.B) {
			for _, count := range []int{32, 128, 8192, 16384, 32768, 65536} {
				b.Run(strconv.Itoa(count), func(b *testing.B) {
					opening, closing := "<code>", "</code>"
					wantCount := 1
					// 表示側のHTMLパーサーは512ノードを越すスタックを拒否する。
					// 走査も同じ失敗時の動作に従い、その本文を書き換えない。
					if count > 512 {
						wantCount = 0
					}
					switch shape {
					case "mismatched":
						closing = "</pre>"
						wantCount = 0
					case "interior":
						opening = "<code><pre>"
					}
					body := strings.Repeat(opening, count) + strings.Repeat(closing, count)
					if shape == "interior" {
						body += strings.Repeat("</pre>", count)
					}
					body += "[[visible]]"
					if got := len(ScanWikilinkMatches(body, "topic")); got != wantCount {
						b.Fatalf("一致の件数 = %d、期待値 = %d", got, wantCount)
					}

					b.ReportAllocs()
					b.SetBytes(int64(len(body)))
					b.ResetTimer()
					for range b.N {
						ScanWikilinkMatches(body, "topic")
					}
				})
			}
		})
	}
}

func BenchmarkScanWikilinkMatchesWithBlockBoundaries(b *testing.B) {
	for _, count := range []int{256, 512, 1024, 2048} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			body := strings.Repeat("> <pre>\n> [[hidden]]\n\n[[visible]]\n\n", count)

			b.ReportAllocs()
			b.SetBytes(int64(len(body)))
			b.ResetTimer()
			for range b.N {
				ScanWikilinkMatches(body, "topic")
			}
		})
	}
}

// BenchmarkScanWithRejectedTokensは、Markdownのコードの中に閉じられていないコメントの
// 開始を持つ本文を測る。HTMLトークナイザーはそれぞれをソース末尾まで届くトークンとして読むため、
// そうしたトークンの内側から読み直す走査は、1つ現れるたびに本文の長さぶんの費用がかかる。
func BenchmarkScanWithRejectedTokens(b *testing.B) {
	for _, count := range []int{1024, 2048, 4096, 8192} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			body := strings.Repeat(
				"`<!--` <code>[[hidden]]</code> [[visible]] ![image](/attachments/01ABC)\n\n",
				count,
			)
			if got := len(ScanWikilinkMatches(body, "topic")); got != count {
				b.Fatalf("Wikiリンクの件数 = %d、期待値 = %d", got, count)
			}
			if got := len(ScanAttachmentRefMatches(body)); got != count {
				b.Fatalf("添付ファイルの参照の件数 = %d、期待値 = %d", got, count)
			}

			b.ReportAllocs()
			b.SetBytes(int64(len(body)))
			b.ResetTimer()
			for range b.N {
				ScanWikilinkMatches(body, "topic")
				ScanAttachmentRefMatches(body)
			}
		})
	}
}

// BenchmarkScanWithRejectedRawTextEndTagsは、エスケープされた終了タグが多数あっても、
// raw textの走査が先頭から再開したり偽の終了タグごとにトークナイザーを作ったりしないことを測る。
func BenchmarkScanWithRejectedRawTextEndTags(b *testing.B) {
	for _, count := range []int{1024, 2048, 4096, 8192} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			body := "x <textarea>" + strings.Repeat("`</textarea>` <object> ", count) +
				"</textarea> [[visible]] <img src=/attachments/01ABC>"
			if got := len(ScanWikilinkMatches(body, "topic")); got != 1 {
				b.Fatalf("Wikiリンクの件数 = %d、期待値 = 1", got)
			}
			if got := len(ScanAttachmentRefMatches(body)); got != 1 {
				b.Fatalf("添付ファイルの件数 = %d、期待値 = 1", got)
			}
			b.ReportAllocs()
			b.SetBytes(int64(len(body)))
			b.ResetTimer()
			for range b.N {
				ScanWikilinkMatches(body, "topic")
				ScanAttachmentRefMatches(body)
			}
		})
	}
}

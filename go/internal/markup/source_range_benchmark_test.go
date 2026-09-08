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
					// Display's HTML parser rejects stacks deeper than 512 nodes. The scan
					// follows the same failure behavior and leaves those bodies unchanged.
					//
					// [Ja] 表示側の HTML パーサーは 512 ノードを越すスタックを拒否する。
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
						b.Fatalf("match count = %d, want %d", got, wantCount)
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

// BenchmarkScanWithRejectedTokens measures a body whose Markdown code holds an opening comment
// with no closing one. The HTML tokenizer reads each of those as a token reaching to the end of
// the source, so a scan that re-reads from inside such a token costs the length of the body every
// time one appears.
//
// [Ja] BenchmarkScanWithRejectedTokens は、Markdown のコードの中に閉じられていないコメントの
// 開始を持つ本文を測る。HTML トークナイザーはそれぞれをソース末尾まで届くトークンとして読むため、
// そうしたトークンの内側から読み直す走査は、1 つ現れるたびに本文の長さぶんの費用がかかる。
func BenchmarkScanWithRejectedTokens(b *testing.B) {
	for _, count := range []int{1024, 2048, 4096, 8192} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			body := strings.Repeat(
				"`<!--` <code>[[hidden]]</code> [[visible]] ![image](/attachments/01ABC)\n\n",
				count,
			)
			if got := len(ScanWikilinkMatches(body, "topic")); got != count {
				b.Fatalf("wiki link count = %d, want %d", got, count)
			}
			if got := len(ScanAttachmentRefMatches(body)); got != count {
				b.Fatalf("attachment reference count = %d, want %d", got, count)
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

// BenchmarkScanWithRejectedRawTextEndTags checks that many escaped end tags do not cause a raw-text
// scan to restart from the beginning or construct a fresh tokenizer for every false closing tag.
//
// [Ja] BenchmarkScanWithRejectedRawTextEndTags は、エスケープされた終了タグが多数あっても、
// raw text の走査が先頭から再開したり偽の終了タグごとにトークナイザーを作ったりしないことを測る。
func BenchmarkScanWithRejectedRawTextEndTags(b *testing.B) {
	for _, count := range []int{1024, 2048, 4096, 8192} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			body := "x <textarea>" + strings.Repeat("`</textarea>` <object> ", count) +
				"</textarea> [[visible]] <img src=/attachments/01ABC>"
			if got := len(ScanWikilinkMatches(body, "topic")); got != 1 {
				b.Fatalf("wiki-link count = %d, want 1", got)
			}
			if got := len(ScanAttachmentRefMatches(body)); got != 1 {
				b.Fatalf("attachment count = %d, want 1", got)
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

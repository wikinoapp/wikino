package draft

import "strings"

// countLineChangesは、本文oldTextをnewTextにする最短の行の編集で、足す行数と消す行数を返す。
// 公開の確認で本文の増減を示すために使う。変えた行は、消す1行と足す1行として数える (`git diff --stat` と同じ)。
func countLineChanges(oldText, newText string) (added, deleted int) {
	a, b := splitLines(oldText), splitLines(newText)

	// 共通の先頭と末尾は編集に関わらないため、比べる範囲から外して計算を減らす。
	for len(a) > 0 && len(b) > 0 && a[0] == b[0] {
		a, b = a[1:], b[1:]
	}
	for len(a) > 0 && len(b) > 0 && a[len(a)-1] == b[len(b)-1] {
		a, b = a[:len(a)-1], b[:len(b)-1]
	}

	// 足す行数と消す行数の和が編集の長さd、差が行数の差になる。
	d := editDistance(a, b)
	return (d + len(b) - len(a)) / 2, (d - len(b) + len(a)) / 2
}

// splitLinesは、textを改行を含めた行に分ける。末尾に改行の無い最終行は、改行のある行と別の行として扱う。
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	lines := strings.SplitAfter(text, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return lines
}

// editDistanceは、aをbにする行の挿入と削除だけの編集の最短の長さを、Myersの差分のアルゴリズムで返す。
// 差分の中身は要らないため、各対角線で到達した位置だけを持ち、メモリを行数に比例する量に抑える。
func editDistance(a, b []string) int {
	n, m := len(a), len(b)
	// 共通部分を除くと追加・削除だけになる場合は、探索すると残りの行数の二乗の時間が掛かる。
	if n == 0 {
		return m
	}
	if m == 0 {
		return n
	}
	limit := n + m

	// v[k+limit] は、対角線k (x-y) で到達したaの位置。
	v := make([]int, 2*limit+2)
	for d := 0; d <= limit; d++ {
		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[k-1+limit] < v[k+1+limit]) {
				x = v[k+1+limit]
			} else {
				x = v[k-1+limit] + 1
			}
			y := x - k
			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}
			v[k+limit] = x
			if x >= n && y >= m {
				return d
			}
		}
	}

	return limit
}

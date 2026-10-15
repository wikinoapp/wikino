package model

import "strings"

// BodyEditは本文の検索置換の1件の編集を表す
type BodyEdit struct {
	// OldStrは置き換える文字列。空にはできない (呼び出し元が形式を検証する)
	OldStr string
	// NewStrは置き換え後の文字列。空なら一致した箇所を削除する
	NewStr string
	// ReplaceAllは一致した箇所をすべて置き換えるかを表す。
	// falseなら一致がちょうど1件の場合だけ置き換える
	ReplaceAll bool
}

// BodyEditMaxBodyBytesは検索置換で各編集を適用した後の本文の最大バイト数。
// 小さな入力から本文を増幅してメモリを枯渇させないよう、途中の結果にも上限を設ける
const BodyEditMaxBodyBytes = 1 << 20

// BodyEditFailureは、本文の検索置換の編集を適用できなかったことを表す
type BodyEditFailure struct {
	// Indexは適用できなかった編集の位置 (0始まり)
	Index int
	// MatchCountは、その編集を適用する時点の本文で `OldStr` に一致した箇所の数。
	// サイズ超過の場合は、一致した箇所の数をそのまま保持する
	MatchCount int
	// SizeExceededは編集後の本文がBodyEditMaxBodyBytesを超えるかを表す
	SizeExceeded bool
}

// ApplyBodyEditsは本文に編集を配列の順に適用し、適用後の本文を返す。
// 各編集は前の編集を適用した結果に対して照合し、大文字小文字を区別して完全に一致する箇所を置き換える。
// 1件でも適用できなければ、適用後の本文を返さずに最初に失敗した編集を返す
// (後の編集は前の編集の結果を前提にするため、失敗した編集より後は照合しない)。
//
// 改行コードの違いで一致しなくならないよう、本文と編集の改行コード (CRLF) をLFに揃えてから照合する。
// そのため、適用後の本文の改行コードはLFになる。
// 各編集後の本文がBodyEditMaxBodyBytesを超える場合は、置換する前に失敗を返す
func ApplyBodyEdits(body string, edits []BodyEdit) (string, *BodyEditFailure) {
	result := normalizeNewlines(body)
	for i, edit := range edits {
		oldStr := normalizeNewlines(edit.OldStr)
		newStr := normalizeNewlines(edit.NewStr)

		if oldStr == "" {
			return "", &BodyEditFailure{Index: i, MatchCount: 0}
		}
		// 重ならない一致を数える。置き換えるstrings.ReplaceAllと同じ数え方にする
		count := strings.Count(result, oldStr)
		if count == 0 || (count > 1 && !edit.ReplaceAll) {
			return "", &BodyEditFailure{Index: i, MatchCount: count}
		}
		// 一致部分の総バイト数は本文以下なので、この積と減算はオーバーフローしない。
		// 追加部分は乗算せず除算で比較し、巨大な置換結果を確保する前に拒否する
		remainingBytes := len(result) - count*len(oldStr)
		if remainingBytes > BodyEditMaxBodyBytes || len(newStr) > (BodyEditMaxBodyBytes-remainingBytes)/count {
			return "", &BodyEditFailure{Index: i, MatchCount: count, SizeExceeded: true}
		}
		result = strings.ReplaceAll(result, oldStr, newStr)
	}
	return result, nil
}

// normalizeNewlinesは改行コードのCRLFをLFに揃える
func normalizeNewlines(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

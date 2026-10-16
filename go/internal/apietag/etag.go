// Package apietagは公開Web APIの条件付きリクエスト (RFC 9110 §13) の `If-Match` と
// `If-None-Match` を、応答の `ETag` と照合する。
// ETagは引用符で囲んだダイジェスト (`"<digest>"`) の強いエンティティタグとして返す前提で扱う。
// 前段のプロキシが圧縮するときに `W/` を付けることがあるため、照合では `W/` の有無を区別しない。
package apietag

import "strings"

// ParseIfMatchは `If-Match` の値 (RFC 9110 §13.1.1) を、照合するダイジェスト (ETagの引用符の中身) の並びに変える。
// 値が `*` なら、リソースが存在すれば一致とするためmatchAnyにtrueを返す。
// 弱いエンティティタグ (`W/`) も、`W/` を外したダイジェストで照合する。RFC 9110の `If-Match` は強い比較
// (§13.1.1) で弱いエンティティタグを一致させないが、ETagは応答の内容から計算したダイジェストのため、
// 途中で `W/` が付いても、ダイジェストが一致すれば内容は同じ版である。
// 引用符で囲まれていないエンティティタグは、どのETagとも一致しないものとして除く
func ParseIfMatch(ifMatch string) (digests []string, matchAny bool) {
	if strings.TrimSpace(ifMatch) == "*" {
		return nil, true
	}
	for tag := range strings.SplitSeq(ifMatch, ",") {
		tag = strings.TrimPrefix(strings.TrimSpace(tag), "W/")
		if len(tag) < 2 || !strings.HasPrefix(tag, `"`) || !strings.HasSuffix(tag, `"`) {
			continue
		}
		digests = append(digests, tag[1:len(tag)-1])
	}
	return digests, false
}

// IfNoneMatchMatchesは `If-None-Match` の値がetagに一致するかを返す (RFC 9110 §13.1.2)。
// 値は `*` か、カンマ区切りのエンティティタグの並び。`If-None-Match` は弱い比較で判定するため、
// `W/` の有無は区別しない
func IfNoneMatchMatches(ifNoneMatch, etag string) bool {
	if strings.TrimSpace(ifNoneMatch) == "*" {
		return true
	}
	for tag := range strings.SplitSeq(ifNoneMatch, ",") {
		if strings.TrimPrefix(strings.TrimSpace(tag), "W/") == etag {
			return true
		}
	}
	return false
}

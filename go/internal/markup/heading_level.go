package markup

import (
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// topHeadingLevelは、本文の最初の見出しを揃える先のレベル。
//
// ページ表示画面はページタイトルをh1として描画しており、本文のh1はそれと同じ階層に並んで
// しまう。本文の見出しをh2から始めることで、画面に出るh1はタイトルの1つだけになり、本文の
// 見出しはその下の階層として飛ばずに読まれる。
const topHeadingLevel = 2

// headingAtomsは見出しのレベル (1〜6) からタグへの対応。添字0は使わない。
var headingAtoms = [...]atom.Atom{0, atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6}

// headingLevelは見出し要素のレベルを返す。見出しでなければ0を返す。
func headingLevel(n *html.Node) int {
	if n.Type != html.ElementNode {
		return 0
	}
	for level := 1; level < len(headingAtoms); level++ {
		if n.DataAtom == headingAtoms[level] {
			return level
		}
	}
	return 0
}

// shiftHeadingsInNodeは、ノード配下の最初の見出しがh2になるよう、すべての見出しを同じ段数
// だけずらす。
//
// 書き始めのレベルに意味を持たせず、書き手が付けた段の差だけを残す。#から書いた本文は1段
// 下がり、##から書いた本文はそのまま、###から書いた本文は1段上がる。
//
// 揃える基準を本文中で最も浅い見出しではなく最初の見出しにするのは、Markdown記法の紹介の
// ように本文の途中に例として#を1つ書いただけで、節の見出しがすべて1段下がるのを避けるため
// である。最初の見出しより浅い見出しはh2に、h6より深くなる見出しはh6に潰す。
//
// Markdownの見出しだけでなく本文に直接書かれたh1なども基準の判定とずらす対象に含めるため、
// Markdownの解析結果ではなく組み立て終わったHTMLツリーに対して行う。差し替えるのはタグ名
// だけで属性は引き継ぐため、見出しのidとそれを指すページ内アンカーは保たれる。
func shiftHeadingsInNode(n *html.Node) {
	firstLevel := firstHeadingLevel(n)
	if firstLevel == 0 {
		return
	}

	// 最初の見出しがh2でずらす段数が0でも、後に現れるh1をh2に潰すために走査は省かない。
	applyHeadingShift(n, topHeadingLevel-firstLevel)
}

// firstHeadingLevelはノード配下を文書順に辿り、最初に現れる見出しのレベルを返す。
// 見出しが無ければ0を返す。
func firstHeadingLevel(n *html.Node) int {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if level := headingLevel(c); level != 0 {
			return level
		}
		if level := firstHeadingLevel(c); level != 0 {
			return level
		}
	}
	return 0
}

// applyHeadingShiftはノード配下の見出しをdeltaだけずらし、h2より浅くなるものはh2に、h6より
// 深くなるものはh6にする。
func applyHeadingShift(n *html.Node, delta int) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if level := headingLevel(c); level != 0 {
			shifted := headingAtoms[min(max(level+delta, topHeadingLevel), len(headingAtoms)-1)]
			c.DataAtom = shifted
			c.Data = shifted.String()
		}

		applyHeadingShift(c, delta)
	}
}

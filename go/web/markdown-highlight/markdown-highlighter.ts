import { tagHighlighter, tags } from "@lezer/highlight";

import { linkDestinationTag, wikilinkTag } from "./markdown-language";

// markdownHighlighterは記法のタグをクラス名に対応付ける。色はstyle.cssに書き、エディタと差分で
// 同じクラス名と色を使う。コードブロックの中身は言語を問わず「コード」の1色にする。
export const markdownHighlighter = tagHighlighter([
  // 記法の記号 (`#`・`**`・`>`・`-`・`[`・`]`・`` ` `` など) とタスク記法・区切り線・コメント
  { tag: [tags.processingInstruction, tags.atom, tags.contentSeparator, tags.comment], class: "wikino-md-mark" },
  // 見出し。heading1〜heading6はheadingの下位のため、まとめて当たる
  { tag: tags.heading, class: "wikino-md-heading" },
  { tag: tags.emphasis, class: "wikino-md-emphasis" },
  { tag: tags.strong, class: "wikino-md-strong" },
  { tag: tags.strikethrough, class: "wikino-md-strikethrough" },
  { tag: tags.link, class: "wikino-md-link" },
  // リンクの中のURLにリンクのクラス名を重ねないよう、URLは別のクラス名にする
  { tag: tags.url, class: "wikino-md-url" },
  { tag: wikilinkTag, class: "wikino-md-link wikino-md-wikilink" },
  // リンクの `(...)` の部分。リンクのテキストの側の `[...]` と色を分ける
  { tag: linkDestinationTag, class: "wikino-md-link-destination" },
  { tag: tags.monospace, class: "wikino-md-code" },
  { tag: tags.quote, class: "wikino-md-quote" },
]);

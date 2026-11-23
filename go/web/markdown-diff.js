// 編集提案の変更画面だけで読み込むエントリ。差分の色付けはCodeMirrorの言語定義を使って大きいため、
// main.jsから分けている。ページの編集画面はeditor.jsが同じモジュールを含むため、1つの画面で
// editor.jsと一緒に読み込まない。
import { initializeMarkdownDiffs } from "./markdown-highlight/markdown-diff";

document.addEventListener("DOMContentLoaded", () => {
  initializeMarkdownDiffs();
});

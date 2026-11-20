// ページの編集画面だけで読み込むエントリ。CodeMirrorを含んで大きいため、エディタを使わない画面の
// main.jsから分けている。main.jsが読み込むモジュールをここから読み込むと、同じモジュールが1つの画面で
// 2つ動いてしまうため読み込まない。
import { initializeDraftConflict } from "./markdown-editor/draft-conflict";
import { initializeDraftDeleteForms } from "./markdown-editor/draft-delete-handler";
import { initializeDraftVersion } from "./markdown-editor/draft-version";
import { initializeEditors } from "./markdown-editor/markdown-editor";
import { initializePublishForms } from "./markdown-editor/publish-handler";
import { initializeSaveStatus } from "./markdown-editor/save-status";
import { initializeMarkdownDiffs } from "./markdown-highlight/markdown-diff";
import { initializeZenMode } from "./zen-mode";

document.addEventListener("DOMContentLoaded", () => {
  initializeEditors();
  initializeDraftDeleteForms();
  initializeSaveStatus();
  initializeDraftVersion();
  initializeDraftConflict();
  initializePublishForms();
  initializeZenMode();
  // 編集履歴のモーダルに読み込む差分に色を付ける
  initializeMarkdownDiffs();
});

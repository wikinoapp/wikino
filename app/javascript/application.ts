import "@hotwired/turbo";
import * as ActiveStorage from "@rails/activestorage";

import { Application } from "@hotwired/stimulus";

// Active Storageの初期化
ActiveStorage.start();

import AbsoluteTimeController from "./controllers/absolute-time-controller";
import AttachmentLoaderController from "./controllers/attachment_loader_controller";
import BulkActionFormController from "./controllers/bulk-action-form-controller";
import editSuggestionDialogController from "./controllers/edit-suggestion-dialog-controller";
import FlashToastController from "./controllers/flash-toast-controller";
import GlobalHotkeyController from "./controllers/global-hotkey-controller";
import MarkdownEditorController from "./controllers/markdown-editor-controller";
import MarkdownEditorFormController from "./controllers/markdown-editor-form-controller";
import SearchCursorController from "./controllers/search-cursor-controller";
import SidebarController from "./controllers/sidebar_controller";
import StuckController from "./controllers/stuck-controller";

declare global {
  interface Window {
    Stimulus: Application;
  }
}

const application = Application.start();
application.debug = false;
window.Stimulus = application;

window.Stimulus.register("absolute-time", AbsoluteTimeController);
window.Stimulus.register("attachment-loader", AttachmentLoaderController);
window.Stimulus.register("bulk-action-form", BulkActionFormController);
window.Stimulus.register("edit-suggestion-dialog", editSuggestionDialogController);
window.Stimulus.register("flash-toast", FlashToastController);
window.Stimulus.register("global-hotkey", GlobalHotkeyController);
window.Stimulus.register("markdown-editor-form", MarkdownEditorFormController);
window.Stimulus.register("markdown-editor", MarkdownEditorController);
window.Stimulus.register("search-cursor", SearchCursorController);
window.Stimulus.register("sidebar", SidebarController);
window.Stimulus.register("stuck", StuckController);

// basecoat-cssのドロップダウンメニューを動的に読み込む
document.addEventListener("turbo:load", () => {
  // 既存のbasecoatスクリプトを削除
  const existingScript = document.querySelector('script[src*="basecoat-css"][src*="dropdown-menu"]');
  if (existingScript) {
    existingScript.remove();
  }

  // 新しいスクリプトタグを作成して追加
  const script = document.createElement("script");
  script.src = "https://cdn.jsdelivr.net/npm/basecoat-css@0.2.8/dist/js/dropdown-menu.min.js";
  script.defer = true;
  document.head.appendChild(script);
});

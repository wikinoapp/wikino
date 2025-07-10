import "@hotwired/turbo";

import { Application } from "@hotwired/stimulus";

import AbsoluteTimeController from "./controllers/absolute-time-controller";
import BulkActionFormController from "./controllers/bulk-action-form-controller";
import FlashToastController from "./controllers/flash-toast-controller";
import MarkdownEditorController from "./controllers/markdown-editor-controller";
import MarkdownEditorFormController from "./controllers/markdown-editor-form-controller";
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
window.Stimulus.register("bulk-action-form", BulkActionFormController);
window.Stimulus.register("flash-toast", FlashToastController);
window.Stimulus.register("markdown-editor-form", MarkdownEditorFormController);
window.Stimulus.register("markdown-editor", MarkdownEditorController);
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

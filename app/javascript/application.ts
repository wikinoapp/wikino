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

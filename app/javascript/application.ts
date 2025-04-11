import "@hotwired/turbo";

import { Application } from "@hotwired/stimulus";

import AbsoluteTimeController from "./controllers/absolute-time-controller";
import BulkActionFormController from "./controllers/bulk-action-form-controller";
import ExportLogsReloaderController from "./controllers/export-logs-reloader-controller";
import ExportStatusWatcherController from "./controllers/export-status-watcher-controller";
import FlashToastController from "./controllers/flash-toast-controller";
import PageEditorController from "./controllers/page-editor-controller";
import PageEditorFormController from "./controllers/page-editor-form-controller";
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
window.Stimulus.register("export-logs-reloader", ExportLogsReloaderController);
window.Stimulus.register("export-status-watcher", ExportStatusWatcherController);
window.Stimulus.register("flash-toast", FlashToastController);
window.Stimulus.register("page-editor-form", PageEditorFormController);
window.Stimulus.register("page-editor", PageEditorController);
window.Stimulus.register("stuck", StuckController);

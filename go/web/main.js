import "basecoat-css/all";

import { initializeAttachmentLoader } from "./attachment-loader";
import { initializeDrawers } from "./drawer";
import { initializeGlobalHotkey } from "./global-hotkey";
import { initializeEditors } from "./markdown-editor/markdown-editor";
import { initializeMarkdownTables } from "./markdown-table";
import { initializePlatform } from "./platform";
import { initializeStickyHeader } from "./sticky-header";
import { initializeZenMode } from "./zen-mode";

window.disableSubmitButtons = function (form) {
  form.querySelectorAll("button[type=submit]").forEach((b) => (b.disabled = true));
};

function setTimeZoneCookie() {
  if (document.cookie.includes("wikino_time_zone=")) return;
  const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
  if (tz) {
    document.cookie = `wikino_time_zone=${tz};path=/;max-age=${60 * 60 * 24 * 365};SameSite=Lax`;
  }
}

document.addEventListener("DOMContentLoaded", () => {
  initializeEditors();
  initializeGlobalHotkey();
  initializeDrawers();
  initializeZenMode();
  initializePlatform();
  initializeAttachmentLoader();
  initializeMarkdownTables();
  initializeStickyHeader();
  setTimeZoneCookie();
});

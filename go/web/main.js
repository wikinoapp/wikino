import "basecoat-css/all";

import { initializeAPITokenScopes } from "./api-token-scope";
import { initializeAttachmentLoader } from "./attachment-loader";
import { initializeCopyableInputs } from "./copyable-input";
import { initializeDrawers } from "./drawer";
import { initializeFormSubmit } from "./form-submit";
import { initializeGlobalHotkey } from "./global-hotkey";
import { initializeDraftConflict } from "./markdown-editor/draft-conflict";
import { initializeDraftDeleteForms } from "./markdown-editor/draft-delete-handler";
import { initializeDraftVersion } from "./markdown-editor/draft-version";
import { initializeEditors } from "./markdown-editor/markdown-editor";
import { initializeMarkdownDiffs } from "./markdown-highlight/markdown-diff";
import { initializePublishForms } from "./markdown-editor/publish-handler";
import { initializeSaveStatus } from "./markdown-editor/save-status";
import { initializeMarkdownTables } from "./markdown-table";
import { initializePlatform } from "./platform";
import { initializeStickyHeader } from "./sticky-header";
import { initializeZenMode } from "./zen-mode";

function setTimeZoneCookie() {
  if (document.cookie.includes("wikino_time_zone=")) return;
  const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
  if (tz) {
    document.cookie = `wikino_time_zone=${tz};path=/;max-age=${60 * 60 * 24 * 365};SameSite=Lax`;
  }
}

document.addEventListener("DOMContentLoaded", () => {
  initializeFormSubmit();
  initializeEditors();
  initializeDraftDeleteForms();
  initializeSaveStatus();
  initializeDraftVersion();
  initializeDraftConflict();
  initializePublishForms();
  initializeGlobalHotkey();
  initializeDrawers();
  initializeZenMode();
  initializePlatform();
  initializeAttachmentLoader();
  initializeAPITokenScopes();
  initializeCopyableInputs();
  initializeMarkdownTables();
  initializeMarkdownDiffs();
  initializeStickyHeader();
  setTimeZoneCookie();
});

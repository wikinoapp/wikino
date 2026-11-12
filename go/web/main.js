import "basecoat-css/all";
// htmxを読み込むのはこのエントリだけにする。機能のエントリ (editor.jsなど) からも読み込むと、
// 1つの画面でhtmxが2つ動くため、それらのエントリはhtmxのAPIが要るときにwindow.htmxを使う
import "htmx.org";

import { initializeAPITokenScopes } from "./api-token-scope";
import { initializeAttachmentLoader } from "./attachment-loader";
import { initializeCopyableInputs } from "./copyable-input";
import { initializeDrawers } from "./drawer";
import { initializeFormSubmit } from "./form-submit";
import { initializeGlobalHotkey } from "./global-hotkey";
import { initializeMarkdownTables } from "./markdown-table";
import { initializePlatform } from "./platform";
import { initializeSearchQuery } from "./search-query";
import { initializeStickyHeader } from "./sticky-header";

function setTimeZoneCookie() {
  if (document.cookie.includes("wikino_time_zone=")) return;
  const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
  if (tz) {
    document.cookie = `wikino_time_zone=${tz};path=/;max-age=${60 * 60 * 24 * 365};SameSite=Lax`;
  }
}

document.addEventListener("DOMContentLoaded", () => {
  initializeFormSubmit();
  initializeGlobalHotkey();
  initializeDrawers();
  initializePlatform();
  initializeAttachmentLoader();
  initializeAPITokenScopes();
  initializeCopyableInputs();
  initializeMarkdownTables();
  initializeStickyHeader();
  initializeSearchQuery();
  setTimeZoneCookie();
});

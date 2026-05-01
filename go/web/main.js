import "basecoat-css/all";

import { initializeGlobalHotkey } from "./global-hotkey";
import { initializeEditors } from "./markdown-editor/markdown-editor";

const SIDEBAR_STORAGE_KEY = "wikinoSidebarOpen";

function initSidebarLocalStoragePersistence() {
  document.addEventListener("basecoat:sidebar", () => {
    const sidebar = document.querySelector(".sidebar");
    if (!sidebar) return;
    const isOpen = sidebar.getAttribute("aria-hidden") === "false";
    localStorage.setItem(SIDEBAR_STORAGE_KEY, String(isOpen));
  });
}

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
  initSidebarLocalStoragePersistence();
  setTimeZoneCookie();
});

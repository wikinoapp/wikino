import "basecoat-css/all";

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

document.addEventListener("DOMContentLoaded", () => {
  initializeEditors();
  initSidebarLocalStoragePersistence();
});

import "basecoat-css/all";

import { initializeEditors } from "./markdown-editor/markdown-editor";

const SIDEBAR_COOKIE_NAME = "wikino_sidebar_open";
const SIDEBAR_COOKIE_MAX_AGE = 365 * 24 * 60 * 60; // 1年

function initSidebarCookiePersistence() {
  const sidebar = document.querySelector(".sidebar");
  if (!sidebar) return;

  document.addEventListener("basecoat:sidebar", () => {
    const isOpen = sidebar.getAttribute("aria-hidden") === "false";
    document.cookie = `${SIDEBAR_COOKIE_NAME}=${isOpen}; path=/; max-age=${SIDEBAR_COOKIE_MAX_AGE}; samesite=lax`;
  });
}

document.addEventListener("DOMContentLoaded", () => {
  initializeEditors();
  initSidebarCookiePersistence();
});

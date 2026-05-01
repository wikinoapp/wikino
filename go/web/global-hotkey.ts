// グローバルホットキー: `s` または `/` キーで検索ページへ遷移する
// Rails 版 (app/javascript/controllers/global-hotkey-controller.ts) と同等の挙動を提供する

const SEARCH_PATH_META_NAME = "wikino-search-path";

export function initializeGlobalHotkey(): void {
  document.addEventListener("keydown", handleKeyDown);
}

function handleKeyDown(event: KeyboardEvent): void {
  if (event.ctrlKey || event.metaKey || event.altKey) {
    return;
  }

  if (event.key !== "s" && event.key !== "/") {
    return;
  }

  if (isInputElement(document.activeElement)) {
    return;
  }

  const searchPath = getMetaContent(SEARCH_PATH_META_NAME);
  if (!searchPath) {
    return;
  }

  event.preventDefault();
  window.location.href = searchPath;
}

function isInputElement(element: Element | null): boolean {
  if (!element) {
    return false;
  }

  const tagName = element.tagName.toLowerCase();
  if (tagName === "input" || tagName === "textarea" || tagName === "select") {
    return true;
  }

  if (element.getAttribute("contenteditable") === "true") {
    return true;
  }

  // CodeMirror エディタの編集領域
  if (element.classList.contains("cm-content")) {
    return true;
  }

  return false;
}

function getMetaContent(name: string): string {
  const meta = document.querySelector<HTMLMetaElement>(`meta[name="${name}"]`);
  return meta?.content ?? "";
}

import { afterEach, beforeAll, beforeEach, describe, expect, it } from "vitest";

import { initializeGlobalHotkey } from "./global-hotkey";

const SEARCH_PATH = "/s/example-space/search";

// global-hotkeyはwindow.location.hrefへの代入で遷移する。happy-domは
// これを実遷移として解決しようとするため、hrefを独自のアクセサで上書きし、
// 代入された値を記録するだけにする。
function stubLocationHref(): () => string | null {
  let assigned: string | null = null;
  Object.defineProperty(window.location, "href", {
    configurable: true,
    get: () => assigned ?? "",
    set: (value: string) => {
      assigned = value;
    },
  });
  return () => assigned;
}

// ホットキーは <meta name="wikino-search-path"> から検索パスを読むため、
// 遷移先を与えるためにmetaを設置する (metaが無いページを再現するときは省く)。
function setSearchPathMeta(path: string): void {
  const meta = document.createElement("meta");
  meta.name = "wikino-search-path";
  meta.content = path;
  document.head.appendChild(meta);
}

function dispatchKey(key: string, init: KeyboardEventInit = {}): KeyboardEvent {
  const event = new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true, ...init });
  document.dispatchEvent(event);
  return event;
}

describe("initializeGlobalHotkey", () => {
  let currentHref: () => string | null;

  beforeAll(() => {
    initializeGlobalHotkey();
  });

  beforeEach(() => {
    document.head.innerHTML = "";
    document.body.innerHTML = "";
    currentHref = stubLocationHref();
  });

  afterEach(() => {
    // 独自のhrefアクセサを外し、次のテストがlocationを新たにスタブし直せるようにする。
    Reflect.deleteProperty(window.location, "href");
  });

  it("'s' キーを押すと検索パスへ遷移する", () => {
    setSearchPathMeta(SEARCH_PATH);

    const event = dispatchKey("s");

    expect(event.defaultPrevented).toBe(true);
    expect(currentHref()).toBe(SEARCH_PATH);
  });

  it("'/' キーを押すと検索パスへ遷移する", () => {
    setSearchPathMeta(SEARCH_PATH);

    const event = dispatchKey("/");

    expect(event.defaultPrevented).toBe(true);
    expect(currentHref()).toBe(SEARCH_PATH);
  });

  // 修飾キー付きの 's' はブラウザ / OSのショートカット (Ctrl+Sの保存など)
  // に属するため、ホットキーは無視しなければならない。
  const modifiers: Array<{ name: string; init: KeyboardEventInit }> = [
    { name: "Ctrl", init: { ctrlKey: true } },
    { name: "Meta", init: { metaKey: true } },
    { name: "Alt", init: { altKey: true } },
  ];

  it.each(modifiers)("修飾キー ($name) と 's' キーを同時に押しても遷移しない", ({ init }) => {
    setSearchPathMeta(SEARCH_PATH);

    const event = dispatchKey("s", init);

    expect(event.defaultPrevented).toBe(false);
    expect(currentHref()).toBeNull();
  });

  it("'s' と '/' 以外のキーを押しても遷移しない", () => {
    setSearchPathMeta(SEARCH_PATH);

    const event = dispatchKey("a");

    expect(event.defaultPrevented).toBe(false);
    expect(currentHref()).toBeNull();
  });

  // 編集・入力中のコンテキストでは 's' や '/' を遷移ではなくそのままの入力
  // として扱わなければならない。
  const focusCases: Array<{ name: string; build: () => HTMLElement }> = [
    { name: "input", build: () => document.createElement("input") },
    { name: "textarea", build: () => document.createElement("textarea") },
    { name: "select", build: () => document.createElement("select") },
    {
      name: "contenteditable",
      build: () => {
        const el = document.createElement("div");
        el.setAttribute("contenteditable", "true");
        el.tabIndex = 0;
        return el;
      },
    },
    {
      name: "CodeMirrorの編集領域",
      build: () => {
        const el = document.createElement("div");
        el.className = "cm-content";
        el.tabIndex = 0;
        return el;
      },
    },
  ];

  it.each(focusCases)("要素 ($name) にフォーカスしている間は遷移しない", ({ build }) => {
    setSearchPathMeta(SEARCH_PATH);
    const el = build();
    document.body.appendChild(el);
    el.focus();

    const event = dispatchKey("s");

    expect(event.defaultPrevented).toBe(false);
    expect(currentHref()).toBeNull();
  });

  it("検索パスのmeta要素が無ければ遷移しない", () => {
    const event = dispatchKey("s");

    expect(event.defaultPrevented).toBe(false);
    expect(currentHref()).toBeNull();
  });
});

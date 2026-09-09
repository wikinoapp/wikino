import { afterEach, beforeAll, beforeEach, describe, expect, it } from "vitest";

import { initializeGlobalHotkey } from "./global-hotkey";

const SEARCH_PATH = "/s/example-space/search";

// global-hotkey navigates by assigning window.location.href. happy-dom would
// try to resolve that as a real navigation, so shadow href with an own accessor
// that only records the assigned value.
//
// [Ja] global-hotkey は window.location.href への代入で遷移する。happy-dom は
// これを実遷移として解決しようとするため、href を独自のアクセサで上書きし、
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

// The hotkey reads the search path from <meta name="wikino-search-path">, so
// install one to give the navigation a destination (omit it to simulate a page
// without the meta).
//
// [Ja] ホットキーは <meta name="wikino-search-path"> から検索パスを読むため、
// 遷移先を与えるために meta を設置する (meta が無いページを再現するときは省く)。
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
    // Drop the own href accessor so the next test re-stubs a fresh location.
    //
    // [Ja] 独自の href アクセサを外し、次のテストが location を新たにスタブし直せるようにする。
    Reflect.deleteProperty(window.location, "href");
  });

  it("navigates to the search path when 's' is pressed", () => {
    setSearchPathMeta(SEARCH_PATH);

    const event = dispatchKey("s");

    expect(event.defaultPrevented).toBe(true);
    expect(currentHref()).toBe(SEARCH_PATH);
  });

  it("navigates to the search path when '/' is pressed", () => {
    setSearchPathMeta(SEARCH_PATH);

    const event = dispatchKey("/");

    expect(event.defaultPrevented).toBe(true);
    expect(currentHref()).toBe(SEARCH_PATH);
  });

  // Modified 's' presses belong to browser/OS shortcuts (Ctrl+S to save etc.),
  // so the hotkey must ignore them.
  //
  // [Ja] 修飾キー付きの 's' はブラウザ / OS のショートカット (Ctrl+S の保存など)
  // に属するため、ホットキーは無視しなければならない。
  const modifiers: Array<{ name: string; init: KeyboardEventInit }> = [
    { name: "Ctrl", init: { ctrlKey: true } },
    { name: "Meta", init: { metaKey: true } },
    { name: "Alt", init: { altKey: true } },
  ];

  it.each(modifiers)("does not navigate when 's' is pressed with $name held", ({ init }) => {
    setSearchPathMeta(SEARCH_PATH);

    const event = dispatchKey("s", init);

    expect(event.defaultPrevented).toBe(false);
    expect(currentHref()).toBeNull();
  });

  it("does not navigate for keys other than 's' or '/'", () => {
    setSearchPathMeta(SEARCH_PATH);

    const event = dispatchKey("a");

    expect(event.defaultPrevented).toBe(false);
    expect(currentHref()).toBeNull();
  });

  // In editable/typing contexts, 's' and '/' must stay literal input instead of
  // triggering navigation.
  //
  // [Ja] 編集・入力中のコンテキストでは 's' や '/' を遷移ではなくそのままの入力
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
      name: "CodeMirror content",
      build: () => {
        const el = document.createElement("div");
        el.className = "cm-content";
        el.tabIndex = 0;
        return el;
      },
    },
  ];

  it.each(focusCases)("does not navigate while a $name element is focused", ({ build }) => {
    setSearchPathMeta(SEARCH_PATH);
    const el = build();
    document.body.appendChild(el);
    el.focus();

    const event = dispatchKey("s");

    expect(event.defaultPrevented).toBe(false);
    expect(currentHref()).toBeNull();
  });

  it("does not navigate when the search path meta is absent", () => {
    const event = dispatchKey("s");

    expect(event.defaultPrevented).toBe(false);
    expect(currentHref()).toBeNull();
  });
});

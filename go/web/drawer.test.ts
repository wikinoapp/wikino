import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { initializeDrawers } from "./drawer";

// Build the DOM that components/drawer.templ renders: an open button that
// references the drawer by id (data-drawer-open), and a drawer that starts
// hidden (hidden class + aria-hidden="true") wrapping a backdrop
// (data-drawer-close). This is the minimal shape initializeDrawers keys off.
//
// [Ja] components/drawer.templ が描画する DOM を組み立てる。id で drawer を参照する
// 開くボタン (data-drawer-open) と、hidden クラス + aria-hidden="true" で閉じた状態から
// 始まり背景 (data-drawer-close) を内包する drawer。initializeDrawers が手がかりにする
// 最小の構造。
function drawerMarkup(id: string): string {
  return `
    <button type="button" data-drawer-open="${id}" aria-controls="${id}" aria-expanded="false">open</button>
    <div id="${id}" class="hidden" data-drawer role="dialog" aria-modal="true" aria-hidden="true">
      <div data-drawer-close></div>
      <div>
        <button type="button" data-drawer-close>close</button>
      </div>
    </div>
  `;
}

function refs(id: string) {
  const opener = document.querySelector(`[data-drawer-open="${id}"]`) as HTMLButtonElement;
  const drawer = document.getElementById(id) as HTMLElement;
  // Two elements carry data-drawer-close: the backdrop overlay (the first one,
  // any element) and the panel's close button (a <button>). Grab them
  // separately so tests can exercise each close entry point.
  //
  // [Ja] data-drawer-close を持つ要素は 2 つある。背景オーバーレイ (最初の要素、任意の
  // 要素) と、パネルの閉じるボタン (<button>)。各閉じる起点を検証できるよう別々に取得する。
  const backdrop = drawer.querySelector("[data-drawer-close]") as HTMLElement;
  const closeButton = drawer.querySelector("button[data-drawer-close]") as HTMLButtonElement;
  return { opener, drawer, backdrop, closeButton };
}

function isOpen(drawer: HTMLElement): boolean {
  return !drawer.classList.contains("hidden") && drawer.getAttribute("aria-hidden") === "false";
}

describe("initializeDrawers", () => {
  beforeAll(() => {
    initializeDrawers();
  });

  beforeEach(() => {
    document.body.innerHTML = "";
  });

  it("opens the matching drawer and reflects it on the open button when the opener is clicked", () => {
    document.body.innerHTML = drawerMarkup("side-drawer");
    const { opener, drawer } = refs("side-drawer");

    opener.click();

    expect(isOpen(drawer)).toBe(true);
    expect(opener.getAttribute("aria-expanded")).toBe("true");
  });

  it("closes the drawer and resets the open button when the backdrop is clicked", () => {
    document.body.innerHTML = drawerMarkup("side-drawer");
    const { opener, drawer, backdrop } = refs("side-drawer");

    opener.click();
    backdrop.click();

    expect(isOpen(drawer)).toBe(false);
    expect(drawer.classList.contains("hidden")).toBe(true);
    expect(drawer.getAttribute("aria-hidden")).toBe("true");
    expect(opener.getAttribute("aria-expanded")).toBe("false");
  });

  it("closes the drawer and resets the open button when the panel close button is clicked", () => {
    document.body.innerHTML = drawerMarkup("side-drawer");
    const { opener, drawer, closeButton } = refs("side-drawer");

    opener.click();
    closeButton.click();

    expect(isOpen(drawer)).toBe(false);
    expect(drawer.classList.contains("hidden")).toBe(true);
    expect(drawer.getAttribute("aria-hidden")).toBe("true");
    expect(opener.getAttribute("aria-expanded")).toBe("false");
  });

  it("closes the open drawer when Escape is pressed", () => {
    document.body.innerHTML = drawerMarkup("side-drawer");
    const { opener, drawer } = refs("side-drawer");

    opener.click();
    expect(isOpen(drawer)).toBe(true);

    document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }));

    expect(isOpen(drawer)).toBe(false);
    expect(drawer.getAttribute("aria-hidden")).toBe("true");
    expect(opener.getAttribute("aria-expanded")).toBe("false");
  });

  it("leaves an already closed drawer untouched on Escape", () => {
    document.body.innerHTML = drawerMarkup("side-drawer");
    const { drawer } = refs("side-drawer");

    document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }));

    expect(isOpen(drawer)).toBe(false);
  });

  it("opens only the drawer whose opener was clicked when several are present", () => {
    document.body.innerHTML = drawerMarkup("first-drawer") + drawerMarkup("second-drawer");
    const first = refs("first-drawer");
    const second = refs("second-drawer");

    first.opener.click();

    expect(isOpen(first.drawer)).toBe(true);
    expect(isOpen(second.drawer)).toBe(false);
  });

  it("follows a drawer added to the DOM after init via event delegation", () => {
    // initializeDrawers is bound once in beforeAll, so a drawer inserted in this
    // test must open without re-binding.
    //
    // [Ja] initializeDrawers は beforeAll で一度だけバインドされるため、このテストで
    // 挿入した drawer も再バインド無しで開かなければならない。

    document.body.innerHTML = drawerMarkup("late-drawer");
    const { opener, drawer } = refs("late-drawer");

    opener.click();

    expect(isOpen(drawer)).toBe(true);
    expect(opener.getAttribute("aria-expanded")).toBe("true");
  });
});

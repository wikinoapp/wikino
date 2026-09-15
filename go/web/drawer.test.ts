import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { initializeDrawers } from "./drawer";

// components/drawer.templが描画するDOMを組み立てる。idでdrawerを参照する
// 開くボタン (data-drawer-open) と、hiddenクラス + aria-hidden="true" で閉じた状態から
// 始まり背景 (data-drawer-close) を内包するdrawer。initializeDrawersが手がかりにする
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
  // data-drawer-closeを持つ要素は2つある。背景オーバーレイ (最初の要素、任意の
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

  it("開くボタンをクリックすると対応するドロワーを開き、ボタンに状態を反映する", () => {
    document.body.innerHTML = drawerMarkup("side-drawer");
    const { opener, drawer } = refs("side-drawer");

    opener.click();

    expect(isOpen(drawer)).toBe(true);
    expect(opener.getAttribute("aria-expanded")).toBe("true");
  });

  it("背景をクリックするとドロワーを閉じ、開くボタンの状態を戻す", () => {
    document.body.innerHTML = drawerMarkup("side-drawer");
    const { opener, drawer, backdrop } = refs("side-drawer");

    opener.click();
    backdrop.click();

    expect(isOpen(drawer)).toBe(false);
    expect(drawer.classList.contains("hidden")).toBe(true);
    expect(drawer.getAttribute("aria-hidden")).toBe("true");
    expect(opener.getAttribute("aria-expanded")).toBe("false");
  });

  it("パネルの閉じるボタンをクリックするとドロワーを閉じ、開くボタンの状態を戻す", () => {
    document.body.innerHTML = drawerMarkup("side-drawer");
    const { opener, drawer, closeButton } = refs("side-drawer");

    opener.click();
    closeButton.click();

    expect(isOpen(drawer)).toBe(false);
    expect(drawer.classList.contains("hidden")).toBe(true);
    expect(drawer.getAttribute("aria-hidden")).toBe("true");
    expect(opener.getAttribute("aria-expanded")).toBe("false");
  });

  it("Escapeキーを押すと開いているドロワーを閉じる", () => {
    document.body.innerHTML = drawerMarkup("side-drawer");
    const { opener, drawer } = refs("side-drawer");

    opener.click();
    expect(isOpen(drawer)).toBe(true);

    document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }));

    expect(isOpen(drawer)).toBe(false);
    expect(drawer.getAttribute("aria-hidden")).toBe("true");
    expect(opener.getAttribute("aria-expanded")).toBe("false");
  });

  it("閉じているドロワーはEscapeキーを押しても変更しない", () => {
    document.body.innerHTML = drawerMarkup("side-drawer");
    const { drawer } = refs("side-drawer");

    document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }));

    expect(isOpen(drawer)).toBe(false);
  });

  it("複数のドロワーがある場合はクリックした開くボタンに対応するものだけを開く", () => {
    document.body.innerHTML = drawerMarkup("first-drawer") + drawerMarkup("second-drawer");
    const first = refs("first-drawer");
    const second = refs("second-drawer");

    first.opener.click();

    expect(isOpen(first.drawer)).toBe(true);
    expect(isOpen(second.drawer)).toBe(false);
  });

  it("初期化後にDOMへ追加したドロワーもイベント委譲で操作できる", () => {
    // initializeDrawersはbeforeAllで一度だけバインドされるため、このテストで
    // 挿入したdrawerも再バインド無しで開かなければならない。

    document.body.innerHTML = drawerMarkup("late-drawer");
    const { opener, drawer } = refs("late-drawer");

    opener.click();

    expect(isOpen(drawer)).toBe(true);
    expect(opener.getAttribute("aria-expanded")).toBe("true");
  });
});

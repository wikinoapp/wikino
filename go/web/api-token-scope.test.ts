import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { initializeAPITokenScopes } from "./api-token-scope";

interface Option {
  scope: string;
  implies?: string[];
  impliedBy?: string[];
  checked?: boolean;
  disabled?: boolean;
  describedBy?: string;
  shownNotes?: string[];
}

function inputID(scope: string): string {
  return `scopes_${scope.replace(":", "_")}`;
}

function noteID(scope: string, implier: string): string {
  return `${inputID(scope)}-implied-by-${implier.replace(":", "_")}`;
}

// components/api_token_scope.templが描画するチェックボックスの並び。
function optionMarkup(option: Option): string {
  const id = inputID(option.scope);
  const notes = (option.impliedBy ?? [])
    .map(
      (implier) => `
        <p id="${noteID(option.scope, implier)}" data-api-token-scope-implied-note="${implier}" ${
          option.shownNotes?.includes(implier) ? "" : "hidden"
        }>「${implier}」に含まれます</p>`,
    )
    .join("");

  return `
    <div class="flex flex-col gap-1">
      <label class="label" for="${id}">
        <input
          class="input"
          id="${id}"
          name="scopes"
          type="checkbox"
          value="${option.scope}"
          data-api-token-scope
          ${option.implies ? `data-api-token-scope-implies="${option.implies.join(" ")}"` : ""}
          ${option.checked ? "checked" : ""}
          ${option.disabled ? "disabled" : ""}
          ${option.describedBy ? `aria-describedby="${option.describedBy}"` : ""}
        />
        ${option.scope}
      </label>
      <p>説明</p>
      ${notes}
    </div>
  `;
}

const DRAFT_PAGE_READ = "draft_page:read";
const DRAFT_PAGE_WRITE = "draft_page:write";
const DRAFT_PAGE_DELETE = "draft_page:delete";

function render(options: { read?: Partial<Option>; write?: Partial<Option>; del?: Partial<Option> } = {}): void {
  document.body.innerHTML = `
    <form>
      ${optionMarkup({ scope: DRAFT_PAGE_READ, impliedBy: [DRAFT_PAGE_WRITE, DRAFT_PAGE_DELETE], ...options.read })}
      ${optionMarkup({ scope: DRAFT_PAGE_WRITE, implies: [DRAFT_PAGE_READ], ...options.write })}
      ${optionMarkup({ scope: DRAFT_PAGE_DELETE, implies: [DRAFT_PAGE_READ], ...options.del })}
    </form>
  `;
}

function checkbox(scope: string): HTMLInputElement {
  return document.getElementById(inputID(scope)) as HTMLInputElement;
}

function note(scope: string, implier: string): HTMLElement {
  return document.getElementById(noteID(scope, implier)) as HTMLElement;
}

// 利用者のクリックと同じく、チェックを切り替えてchangeを送る。
function toggle(scope: string): void {
  checkbox(scope).click();
}

describe("initializeAPITokenScopes", () => {
  beforeAll(() => {
    initializeAPITokenScopes();
  });

  beforeEach(() => {
    render();
  });

  it("含むスコープを選ぶと、含まれるスコープを選択済みにして無効にし、注記を出す", () => {
    toggle(DRAFT_PAGE_WRITE);

    const read = checkbox(DRAFT_PAGE_READ);
    expect(read.checked).toBe(true);
    expect(read.disabled).toBe(true);
    expect(note(DRAFT_PAGE_READ, DRAFT_PAGE_WRITE).hidden).toBe(false);
    expect(note(DRAFT_PAGE_READ, DRAFT_PAGE_DELETE).hidden).toBe(true);
    expect(read.getAttribute("aria-describedby")).toBe(noteID(DRAFT_PAGE_READ, DRAFT_PAGE_WRITE));
  });

  it("含むスコープを外すと、選んでいなかった状態に戻す", () => {
    toggle(DRAFT_PAGE_WRITE);
    toggle(DRAFT_PAGE_WRITE);

    const read = checkbox(DRAFT_PAGE_READ);
    expect(read.checked).toBe(false);
    expect(read.disabled).toBe(false);
    expect(note(DRAFT_PAGE_READ, DRAFT_PAGE_WRITE).hidden).toBe(true);
    expect(read.hasAttribute("aria-describedby")).toBe(false);
  });

  it("含まれる前に選んでいたスコープは、含むスコープを外しても選んだままにする", () => {
    toggle(DRAFT_PAGE_READ);
    toggle(DRAFT_PAGE_WRITE);
    toggle(DRAFT_PAGE_WRITE);

    const read = checkbox(DRAFT_PAGE_READ);
    expect(read.checked).toBe(true);
    expect(read.disabled).toBe(false);
  });

  it("含むスコープが複数選ばれているあいだは、すべてを外すまで無効のままにする", () => {
    toggle(DRAFT_PAGE_WRITE);
    toggle(DRAFT_PAGE_DELETE);

    const read = checkbox(DRAFT_PAGE_READ);
    expect(note(DRAFT_PAGE_READ, DRAFT_PAGE_WRITE).hidden).toBe(false);
    expect(note(DRAFT_PAGE_READ, DRAFT_PAGE_DELETE).hidden).toBe(false);
    expect(read.getAttribute("aria-describedby")).toBe(
      `${noteID(DRAFT_PAGE_READ, DRAFT_PAGE_WRITE)} ${noteID(DRAFT_PAGE_READ, DRAFT_PAGE_DELETE)}`,
    );

    toggle(DRAFT_PAGE_WRITE);
    expect(read.checked).toBe(true);
    expect(read.disabled).toBe(true);
    expect(note(DRAFT_PAGE_READ, DRAFT_PAGE_WRITE).hidden).toBe(true);
    expect(read.getAttribute("aria-describedby")).toBe(noteID(DRAFT_PAGE_READ, DRAFT_PAGE_DELETE));

    toggle(DRAFT_PAGE_DELETE);
    expect(read.checked).toBe(false);
    expect(read.disabled).toBe(false);
  });

  it("サーバーが含まれる状態で描画したスコープは、含むスコープを外すと選んでいない状態に戻す", () => {
    const describedBy = `scopes-error ${noteID(DRAFT_PAGE_READ, DRAFT_PAGE_WRITE)}`;
    render({
      read: { checked: true, disabled: true, describedBy, shownNotes: [DRAFT_PAGE_WRITE] },
      write: { checked: true },
    });

    toggle(DRAFT_PAGE_WRITE);

    const read = checkbox(DRAFT_PAGE_READ);
    expect(read.checked).toBe(false);
    expect(read.disabled).toBe(false);
    expect(note(DRAFT_PAGE_READ, DRAFT_PAGE_WRITE).hidden).toBe(true);
    // 選択のエラーなど、含まれることの注記以外の参照は残す。
    expect(read.getAttribute("aria-describedby")).toBe("scopes-error");
  });

  it("別の名前のチェックボックスの並びには影響しない", () => {
    document.body.insertAdjacentHTML(
      "beforeend",
      `<form>${optionMarkup({ scope: DRAFT_PAGE_READ, impliedBy: [DRAFT_PAGE_WRITE] })
        .replace('name="scopes"', 'name="granted_scopes"')
        .replace(`id="${inputID(DRAFT_PAGE_READ)}"`, 'id="other_read"')}</form>`,
    );

    toggle(DRAFT_PAGE_WRITE);

    const other = document.getElementById("other_read") as HTMLInputElement;
    expect(other.checked).toBe(false);
    expect(other.disabled).toBe(false);
  });
});

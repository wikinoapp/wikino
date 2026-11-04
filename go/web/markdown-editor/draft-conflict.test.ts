import { afterEach, describe, expect, it, vi } from "vitest";

import { initializeDraftConflict } from "./draft-conflict";

vi.mock("./markdown-editor", () => ({ overwriteConflictedDraft: vi.fn() }));
vi.mock("./publish-handler", () => ({ republishConflictedDraft: vi.fn() }));

// pages/page/edit.templの競合の知らせを写す
function conflictMarkup(publishConflict: boolean): string {
  const attributes = publishConflict ? 'data-draft-conflict-publish tabindex="-1"' : "hidden";
  return `
    <input id="page_title" autofocus>
    <div data-draft-conflict ${attributes}>
      <div role="alert">下書きが別の場所で更新されています</div>
    </div>
  `;
}

describe("initializeDraftConflict", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  it("公開が競合して再表示した編集画面では、知らせへフォーカスを移す", () => {
    document.body.innerHTML = conflictMarkup(true);
    document.querySelector<HTMLInputElement>("#page_title")!.focus();

    initializeDraftConflict();

    expect(document.activeElement).toBe(document.querySelector("[data-draft-conflict]"));
  });

  it("競合していない編集画面では、フォーカスを動かさない", () => {
    document.body.innerHTML = conflictMarkup(false);
    const title = document.querySelector<HTMLInputElement>("#page_title")!;
    title.focus();

    initializeDraftConflict();

    expect(document.activeElement).toBe(title);
  });
});

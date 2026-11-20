import { afterEach, describe, expect, it } from "vitest";

import { placeCursorAfterSpaceFilters } from "./search-query";

function createInput(value: string): HTMLInputElement {
  const input = document.createElement("input");
  input.type = "text";
  input.value = value;
  document.body.appendChild(input);
  return input;
}

describe("placeCursorAfterSpaceFilters", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  it("space:の指定だけなら、後ろに空白を足してカーソルを置く", () => {
    const input = createInput("space:wiki");

    placeCursorAfterSpaceFilters(input);

    expect(input.value).toBe("space:wiki ");
    expect(input.selectionStart).toBe(11);
    expect(input.selectionEnd).toBe(11);
  });

  it("指定の後ろに空白があれば、足さずにその後ろへ置く", () => {
    const input = createInput("space:wiki space:docs ");

    placeCursorAfterSpaceFilters(input);

    expect(input.value).toBe("space:wiki space:docs ");
    expect(input.selectionStart).toBe(22);
  });

  it("キーワードがあれば、カーソルを先頭に置く", () => {
    const input = createInput("space:wiki daily");

    placeCursorAfterSpaceFilters(input);

    expect(input.value).toBe("space:wiki daily");
    expect(input.selectionStart).toBe(0);
  });

  it("space:の指定が無ければ、何もしない", () => {
    const input = createInput("日報");
    input.setSelectionRange(2, 2);

    placeCursorAfterSpaceFilters(input);

    expect(input.value).toBe("日報");
    expect(input.selectionStart).toBe(2);
  });
});

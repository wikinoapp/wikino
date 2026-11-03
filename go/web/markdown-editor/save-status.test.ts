import { afterEach, beforeAll, beforeEach, describe, expect, it } from "vitest";

import { clearSaveError, initializeSaveStatus, showSaveError } from "./save-status";

// pages/page/edit.templの保存の表示と、下書き保存のボタンを写す
function statusMarkup(): string {
  return `
    <button type="button" id="save-draft" data-draft-save-button>Save draft</button>
    <button type="button" id="preview">Preview</button>
    <div
      data-draft-save-status
      data-autosave-failed-message="自動保存できませんでした。"
      data-save-draft-failed-message="下書きを保存できませんでした。"
      data-body-too-large-message="本文が長すぎます。本文は500,000文字以内にしてください。"
    >
      <div id="page-draft-saved-at">自動保存: 12:00</div>
      <p id="page-draft-save-error" role="status" data-draft-save-error></p>
    </div>
  `;
}

function errorText(): string {
  return document.querySelector("[data-draft-save-error]")?.textContent ?? "";
}

// htmx 4はリクエストの元の要素で、応答の状態と本文をctxに入れてhtmx:after:requestを発火する
function dispatchAfterRequest(elementId: string, status: number, text: string): void {
  document
    .getElementById(elementId)
    ?.dispatchEvent(
      new CustomEvent("htmx:after:request", { bubbles: true, detail: { ctx: { response: { status }, text } } }),
    );
}

describe("showSaveError", () => {
  beforeEach(() => {
    document.body.innerHTML = statusMarkup();
  });

  afterEach(() => {
    document.body.innerHTML = "";
  });

  it("422のときはサーバーが返した文面を、自動保存の失敗として出す", () => {
    const shown = showSaveError("autosave", 422, "本文は500,000文字以内で入力してください (現在500,001文字)\n");

    expect(shown).toBe(true);
    expect(errorText()).toBe("自動保存できませんでした。 本文は500,000文字以内で入力してください (現在500,001文字)");
  });

  it("413のときは応答の本文を使わず、本文が長すぎる旨を出す", () => {
    showSaveError("manual", 413, "<!doctype html><title>Content Too Large</title>");

    expect(errorText()).toBe("下書きを保存できませんでした。 本文が長すぎます。本文は500,000文字以内にしてください。");
  });

  it("利用者が直せない失敗では表示を変えない", () => {
    expect(showSaveError("autosave", 500, "Internal Server Error")).toBe(false);
    expect(showSaveError("autosave", 403, "Forbidden")).toBe(false);

    expect(errorText()).toBe("");
  });

  it("文面が同じときは書き直さず、読み上げを繰り返さない", () => {
    showSaveError("autosave", 413, "");
    const textNode = document.querySelector("[data-draft-save-error]")?.firstChild;

    showSaveError("autosave", 413, "");

    expect(document.querySelector("[data-draft-save-error]")?.firstChild).toBe(textNode);
  });

  it("保存の表示が無い画面では何もしない", () => {
    document.body.innerHTML = "";

    expect(showSaveError("autosave", 422, "本文が長すぎます")).toBe(false);
  });
});

describe("clearSaveError", () => {
  beforeEach(() => {
    document.body.innerHTML = statusMarkup();
  });

  afterEach(() => {
    document.body.innerHTML = "";
  });

  it("保存に成功したらエラーを消す", () => {
    showSaveError("autosave", 413, "");

    clearSaveError();

    expect(errorText()).toBe("");
  });
});

describe("initializeSaveStatus", () => {
  beforeAll(() => {
    // リスナーはdocumentに委譲するため、テスト全体で1度だけ登録する
    initializeSaveStatus();
  });

  beforeEach(() => {
    document.body.innerHTML = statusMarkup();
  });

  afterEach(() => {
    document.body.innerHTML = "";
  });

  it("下書き保存が422で失敗したら、下書き保存の失敗として出す", () => {
    dispatchAfterRequest("save-draft", 422, "本文は500,000文字以内で入力してください (現在500,001文字)");

    expect(errorText()).toBe(
      "下書きを保存できませんでした。 本文は500,000文字以内で入力してください (現在500,001文字)",
    );
  });

  it("下書き保存が413で失敗したら、本文が長すぎる旨を出す", () => {
    dispatchAfterRequest("save-draft", 413, "<div>fragment</div>");

    expect(errorText()).toBe("下書きを保存できませんでした。 本文が長すぎます。本文は500,000文字以内にしてください。");
  });

  it("下書き保存に成功したらエラーを消す", () => {
    dispatchAfterRequest("save-draft", 422, "本文が長すぎます");

    dispatchAfterRequest("save-draft", 204, "");

    expect(errorText()).toBe("");
  });

  it("下書き保存のボタン以外の要求には反応しない", () => {
    dispatchAfterRequest("preview", 413, "<div>fragment</div>");

    expect(errorText()).toBe("");
  });
});

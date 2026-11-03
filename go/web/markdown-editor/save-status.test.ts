import { afterEach, beforeAll, beforeEach, describe, expect, it } from "vitest";

import { beginDraftSave, clearSaveError, initializeSaveStatus, showSaveError } from "./save-status";

// pages/page/edit.templの保存の表示と、下書き保存のボタンを写す
function statusMarkup(): string {
  return `
    <input id="page_title" value="Title">
    <textarea id="page_body">Body</textarea>
    <div data-markdown-editor data-markdown-editor-title="#page_title" data-markdown-editor-textarea="#page_body"></div>
    <button type="button" id="save-draft" data-draft-save-button>Save draft</button>
    <button type="button" id="preview">Preview</button>
    <div id="page-draft-refresh-trigger"></div>
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

interface RequestContext {
  request: { body: FormData | null };
  response?: { status: number };
  text?: string;
}

function formData(): FormData {
  const data = new FormData();
  data.set("title", document.querySelector<HTMLInputElement>("#page_title")?.value ?? "");
  data.set("body", document.querySelector<HTMLTextAreaElement>("#page_body")?.value ?? "");
  return data;
}

// alpha8と同じctxを要求の開始から応答まで使い、after:requestの取り消しも観測する。
function beginRequest(elementId = "save-draft", data: FormData | null = formData()): RequestContext {
  const ctx: RequestContext = { request: { body: data } };
  document
    .getElementById(elementId)
    ?.dispatchEvent(new CustomEvent("htmx:before:request", { bubbles: true, cancelable: true, detail: { ctx } }));
  return ctx;
}

function finishRequest(ctx: RequestContext, status: number, text: string, elementId = "save-draft"): Event {
  ctx.response = { status };
  ctx.text = text;
  const event = new CustomEvent("htmx:after:request", { bubbles: true, cancelable: true, detail: { ctx } });
  document.getElementById(elementId)?.dispatchEvent(event);
  return event;
}

function dispatchAfterRequest(elementId: string, status: number, text: string): void {
  finishRequest(beginRequest(elementId), status, text, elementId);
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

  it("新しい手動保存の失敗のあとに古い成功が届いても、エラーとOOB表示を戻さない", () => {
    const old = beginRequest();
    (document.querySelector("#page_body") as HTMLTextAreaElement).value = "New body";
    dispatchAfterRequest("save-draft", 422, "新しい本文が長すぎます");

    const event = finishRequest(old, 200, '<div id="page-draft-saved-at" hx-swap-oob="outerHTML">old</div>');

    expect(errorText()).toContain("新しい本文が長すぎます");
    expect(event.defaultPrevented).toBe(true);
  });

  it("新しい成功のあとに古い手動保存の失敗を表示しない", () => {
    const old = beginRequest();
    (document.querySelector("#page_body") as HTMLTextAreaElement).value = "New body";
    dispatchAfterRequest("save-draft", 204, "");

    const event = finishRequest(old, 422, "古い本文が長すぎます");

    expect(errorText()).toBe("");
    expect(event.defaultPrevented).toBe(true);
  });

  it("次の送信前でも、タイトルが変わっていれば古い成功による画面遷移を止める", () => {
    const old = beginRequest();
    (document.querySelector("#page_title") as HTMLInputElement).value = "New title";

    const event = finishRequest(old, 204, "");

    expect(event.defaultPrevented).toBe(true);
  });

  it("キューで待っていた手動保存は、組み立て済みの送信内容で照合する", () => {
    const queued = formData();
    (document.querySelector("#page_body") as HTMLTextAreaElement).value = "New body";

    const event = finishRequest(beginRequest("save-draft", queued), 422, "古い本文が長すぎます");

    expect(errorText()).toBe("");
    expect(event.defaultPrevented).toBe(true);
  });

  it("保存時刻の再取得中に新しい保存が始まったら、古い再取得のOOBスワップを止める", () => {
    const container = document.querySelector<HTMLElement>("[data-markdown-editor]")!;
    beginDraftSave(container, formData());
    const refresh = beginRequest("page-draft-refresh-trigger", null);
    (document.querySelector("#page_body") as HTMLTextAreaElement).value = "New body";
    dispatchAfterRequest("save-draft", 422, "新しい本文が長すぎます");

    const event = finishRequest(refresh, 200, "saved time", "page-draft-refresh-trigger");

    expect(event.defaultPrevented).toBe(true);
    expect(errorText()).toContain("新しい本文が長すぎます");
  });

  it("現在の保存に対する再取得は通し、保存の成否は書き換えない", () => {
    const container = document.querySelector<HTMLElement>("[data-markdown-editor]")!;
    beginDraftSave(container, formData());
    const refresh = beginRequest("page-draft-refresh-trigger", null);

    const event = finishRequest(refresh, 200, "saved time", "page-draft-refresh-trigger");

    expect(event.defaultPrevented).toBe(false);
    expect(errorText()).toBe("");
  });
});

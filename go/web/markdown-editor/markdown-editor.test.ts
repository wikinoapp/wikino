import { EditorView } from "codemirror";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { initializeEditors, stopAutosave } from "./markdown-editor";
import { initializeSaveStatus } from "./save-status";

interface MarkdownEditorContainer extends HTMLElement {
  _editorView?: EditorView;
}

// pages/page/edit.templの本文エディタを写す。エディタの命名元になる可視ラベル (CodeMirrorの
// contenteditableはlabelの関連先になれないため `for` は持たない)、data-* 設定を持つコンテナ、
// 値を保持する非表示のtextarea。
function editorMarkup(): string {
  return `
    <label id="page-body-label">Body</label>
    <input id="page_title" value="Title">
    <div
      data-markdown-editor
      data-markdown-editor-label="#page-body-label"
      data-markdown-editor-textarea="#page_body"
      data-markdown-editor-title="#page_title"
      data-markdown-editor-body="Page body"
      data-markdown-editor-space-identifier="test-space"
    ></div>
    <textarea id="page_body">Page body</textarea>
  `;
}

describe("initializeEditors", () => {
  beforeEach(() => {
    document.body.innerHTML = editorMarkup();
  });

  afterEach(() => {
    const container = document.querySelector<MarkdownEditorContainer>("[data-markdown-editor]");
    container?._editorView?.destroy();
    document.body.innerHTML = "";
  });

  it("表示されている本文ラベルをCodeMirrorのテキストボックスの名前に使う", () => {
    initializeEditors();

    const textbox = document.querySelector<HTMLElement>('.cm-content[role="textbox"]');

    expect(textbox?.getAttribute("aria-labelledby")).toBe("page-body-label");
  });

  it("表示されている本文ラベルをクリックするとCodeMirrorのテキストボックスにフォーカスする", () => {
    initializeEditors();

    const label = document.getElementById("page-body-label") as HTMLLabelElement;
    const textbox = document.querySelector<HTMLElement>('.cm-content[role="textbox"]');

    label.click();

    expect(document.activeElement).toBe(textbox);
  });

  it("ラベルのセレクタが無くてもエディタを初期化する", () => {
    const container = document.querySelector<MarkdownEditorContainer>("[data-markdown-editor]");
    container?.removeAttribute("data-markdown-editor-label");

    initializeEditors();

    const textbox = document.querySelector<HTMLElement>('.cm-content[role="textbox"]');

    expect(textbox).not.toBeNull();
    expect(textbox?.hasAttribute("aria-labelledby")).toBe(false);
  });

  it("ラベルのセレクタに一致する要素が無くてもエディタを初期化する", () => {
    const container = document.querySelector<MarkdownEditorContainer>("[data-markdown-editor]");
    container?.setAttribute("data-markdown-editor-label", "#renamed-body-label");

    initializeEditors();

    const textbox = document.querySelector<HTMLElement>('.cm-content[role="textbox"]');

    expect(textbox).not.toBeNull();
    expect(textbox?.hasAttribute("aria-labelledby")).toBe(false);
  });

  it("一致したラベルにidが無ければaria-labelledbyを付けない", () => {
    const container = document.querySelector<MarkdownEditorContainer>("[data-markdown-editor]");
    const label = document.getElementById("page-body-label") as HTMLLabelElement;
    label.removeAttribute("id");
    label.className = "label";
    container?.setAttribute("data-markdown-editor-label", ".label");

    initializeEditors();

    const textbox = document.querySelector<HTMLElement>('.cm-content[role="textbox"]');

    expect(textbox).not.toBeNull();
    expect(textbox?.hasAttribute("aria-labelledby")).toBe(false);
  });

  it("本文のエラーを説明する要素をCodeMirrorのテキストボックスに関連付ける", () => {
    const container = document.querySelector<MarkdownEditorContainer>("[data-markdown-editor]");
    container?.setAttribute("data-markdown-editor-describedby", "page-body-error page-draft-save-error");
    container?.setAttribute("data-markdown-editor-invalid", "true");

    initializeEditors();

    const textbox = document.querySelector<HTMLElement>('.cm-content[role="textbox"]');

    expect(textbox?.getAttribute("aria-describedby")).toBe("page-body-error page-draft-save-error");
    expect(textbox?.getAttribute("aria-invalid")).toBe("true");
  });

  it("本文のエラーが無ければaria-invalidを付けない", () => {
    const container = document.querySelector<MarkdownEditorContainer>("[data-markdown-editor]");
    container?.setAttribute("data-markdown-editor-invalid", "false");

    initializeEditors();

    const textbox = document.querySelector<HTMLElement>('.cm-content[role="textbox"]');

    expect(textbox?.hasAttribute("aria-invalid")).toBe(false);
  });
});

describe("stopAutosave", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    vi.useFakeTimers();
    fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);

    document.body.innerHTML = editorMarkup();
    document
      .querySelector("[data-markdown-editor]")
      ?.setAttribute("data-markdown-editor-draft-save-url", "/s/test-space/pages/1/draft_page");
    initializeEditors();
  });

  afterEach(() => {
    const container = document.querySelector<MarkdownEditorContainer>("[data-markdown-editor]");
    container?._editorView?.destroy();
    document.body.innerHTML = "";
    vi.unstubAllGlobals();
    vi.useRealTimers();
  });

  function typeIntoEditor(text: string): void {
    const view = document.querySelector<MarkdownEditorContainer>("[data-markdown-editor]")?._editorView;
    view?.dispatch({ changes: { from: 0, insert: text } });
  }

  it("止める前の入力による自動保存の予約を取り消す", async () => {
    typeIntoEditor("a");

    await stopAutosave();
    await vi.advanceTimersByTimeAsync(1000);

    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("止めた後の入力では自動保存しない", async () => {
    await stopAutosave();

    typeIntoEditor("a");
    await vi.advanceTimersByTimeAsync(1000);

    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("送信中の自動保存が終わるまで解決しない", async () => {
    let resolveFetch: (response: Response) => void = () => {};
    fetchMock.mockReturnValue(new Promise<Response>((resolve) => (resolveFetch = resolve)));

    typeIntoEditor("a");
    await vi.advanceTimersByTimeAsync(1000);
    expect(fetchMock).toHaveBeenCalledTimes(1);

    let stopped = false;
    const stopping = stopAutosave().then(() => (stopped = true));

    await vi.advanceTimersByTimeAsync(0);
    expect(stopped).toBe(false);

    resolveFetch(new Response(null, { status: 200 }));
    await stopping;
    expect(stopped).toBe(true);
  });

  it("重複した自動保存はすべて終わるまで解決しない", async () => {
    let resolveFirst: (response: Response) => void = () => {};
    let resolveSecond: (response: Response) => void = () => {};
    fetchMock
      .mockImplementationOnce(() => new Promise<Response>((resolve) => (resolveFirst = resolve)))
      .mockImplementationOnce(() => new Promise<Response>((resolve) => (resolveSecond = resolve)));

    typeIntoEditor("a");
    await vi.advanceTimersByTimeAsync(500);
    typeIntoEditor("b");
    await vi.advanceTimersByTimeAsync(500);
    expect(fetchMock).toHaveBeenCalledTimes(2);

    let stopped = false;
    const stopping = stopAutosave().then(() => (stopped = true));

    resolveSecond(new Response(null, { status: 200 }));
    await vi.advanceTimersByTimeAsync(0);
    expect(stopped).toBe(false);

    resolveFirst(new Response(null, { status: 200 }));
    await stopping;
    expect(stopped).toBe(true);
  });

  it("破棄したエディタの送信中の保存は待たない", async () => {
    fetchMock.mockReturnValue(new Promise<Response>(() => {}));

    typeIntoEditor("a");
    await vi.advanceTimersByTimeAsync(1000);
    expect(fetchMock).toHaveBeenCalledTimes(1);

    document.querySelector<MarkdownEditorContainer>("[data-markdown-editor]")?._editorView?.destroy();

    await expect(stopAutosave()).resolves.toBeUndefined();
  });

  it("送信中の自動保存が失敗しても解決する", async () => {
    let rejectFetch: (error: Error) => void = () => {};
    fetchMock.mockReturnValue(new Promise<Response>((_, reject) => (rejectFetch = reject)));

    typeIntoEditor("a");
    await vi.advanceTimersByTimeAsync(1000);

    const stopping = stopAutosave();
    rejectFetch(new Error("network error"));

    await expect(stopping).resolves.toBeUndefined();
  });
});

// pages/page/edit.templの保存の表示を写す
function saveStatusMarkup(): string {
  return `
    <button type="button" id="save-draft" data-draft-save-button>Save draft</button>
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

describe("自動保存の失敗の表示", () => {
  let fetchMock: ReturnType<typeof vi.fn>;
  let autosaved = vi.fn<() => void>();

  beforeAll(() => initializeSaveStatus());

  beforeEach(() => {
    vi.useFakeTimers();
    fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
    autosaved = vi.fn();
    window.addEventListener("draft-autosaved", autosaved);

    document.body.innerHTML = editorMarkup() + saveStatusMarkup();
    document
      .querySelector("[data-markdown-editor]")
      ?.setAttribute("data-markdown-editor-draft-save-url", "/s/test-space/pages/1/draft_page");
    document
      .querySelector("[data-markdown-editor]")
      ?.setAttribute("data-markdown-editor-describedby", "page-draft-save-error");
    initializeEditors();
  });

  afterEach(() => {
    const container = document.querySelector<MarkdownEditorContainer>("[data-markdown-editor]");
    container?._editorView?.destroy();
    window.removeEventListener("draft-autosaved", autosaved);
    document.body.innerHTML = "";
    vi.unstubAllGlobals();
    vi.useRealTimers();
  });

  function typeIntoEditor(text: string): void {
    const view = document.querySelector<MarkdownEditorContainer>("[data-markdown-editor]")?._editorView;
    view?.dispatch({ changes: { from: 0, insert: text } });
  }

  function errorText(): string {
    return document.querySelector("[data-draft-save-error]")?.textContent ?? "";
  }

  function textbox(): HTMLElement {
    return document.querySelector<HTMLElement>('.cm-content[role="textbox"]')!;
  }

  async function pendingAutosave(): Promise<(response: Response) => void> {
    const { promise, resolve } = Promise.withResolvers<Response>();
    fetchMock.mockReturnValueOnce(promise);
    typeIntoEditor("a");
    await vi.advanceTimersByTimeAsync(1000);
    return resolve;
  }

  function beginManualSave() {
    const body = new FormData();
    body.set("title", document.querySelector<HTMLInputElement>("#page_title")!.value);
    body.set("body", document.querySelector<HTMLTextAreaElement>("#page_body")!.value);
    const ctx = { request: { body }, response: { status: 0 }, text: "" };
    document
      .getElementById("save-draft")!
      .dispatchEvent(new CustomEvent("htmx:before:request", { bubbles: true, cancelable: true, detail: { ctx } }));
    return (status: number, text: string): Event => {
      ctx.response.status = status;
      ctx.text = text;
      const event = new CustomEvent("htmx:after:request", { bubbles: true, cancelable: true, detail: { ctx } });
      document.getElementById("save-draft")!.dispatchEvent(event);
      return event;
    };
  }

  function initializeWithPublishError(): void {
    const container = document.querySelector<MarkdownEditorContainer>("[data-markdown-editor]")!;
    container._editorView?.destroy();
    container.dataset.markdownEditorInvalid = "true";
    container.dataset.markdownEditorDescribedby = "page-body-error page-draft-save-error";
    document.body.insertAdjacentHTML("beforeend", '<div id="page-body-error">本文が長すぎます</div>');
    initializeEditors();
  }

  async function autosave(response: Response): Promise<void> {
    fetchMock.mockResolvedValueOnce(response);
    typeIntoEditor("a");
    await vi.advanceTimersByTimeAsync(1000);
  }

  it("422で失敗したら、返った文面を自動保存の表示の位置に出す", async () => {
    await autosave(new Response("本文は500,000文字以内で入力してください (現在500,001文字)", { status: 422 }));

    expect(errorText()).toBe("自動保存できませんでした。 本文は500,000文字以内で入力してください (現在500,001文字)");
  });

  it("413で失敗したら、本文が長すぎる旨を出す", async () => {
    await autosave(new Response("<!doctype html><title>Content Too Large</title>", { status: 413 }));

    expect(errorText()).toBe("自動保存できませんでした。 本文が長すぎます。本文は500,000文字以内にしてください。");
  });

  it("保存できたらエラーを消し、保存時刻の再取得を始める", async () => {
    await autosave(new Response("本文が長すぎます", { status: 422 }));
    expect(autosaved).not.toHaveBeenCalled();

    await autosave(new Response(null, { status: 204 }));

    expect(errorText()).toBe("");
    expect(autosaved).toHaveBeenCalledTimes(1);
  });

  it("利用者が直せない失敗では表示を変えない", async () => {
    await autosave(new Response("Internal Server Error", { status: 500 }));

    expect(errorText()).toBe("");
  });

  it("新しい自動保存の失敗を、遅れて届いた古い成功が消さない", async () => {
    const resolveOld = await pendingAutosave();
    await autosave(new Response("新しい本文が長すぎます", { status: 422 }));

    resolveOld(new Response(null, { status: 204 }));
    await vi.advanceTimersByTimeAsync(0);

    expect(errorText()).toContain("新しい本文が長すぎます");
    expect(textbox().getAttribute("aria-invalid")).toBe("true");
    expect(autosaved).not.toHaveBeenCalled();
  });

  it("新しい自動保存の成功後に、古い失敗を出さない", async () => {
    const resolveOld = await pendingAutosave();
    await autosave(new Response(null, { status: 204 }));

    resolveOld(new Response("古い本文が長すぎます", { status: 422 }));
    await vi.advanceTimersByTimeAsync(0);

    expect(errorText()).toBe("");
    expect(textbox().hasAttribute("aria-invalid")).toBe(false);
    expect(autosaved).toHaveBeenCalledTimes(1);
  });

  it("次の自動保存の送信前でも、入力後に届いた古い成功を表示しない", async () => {
    const resolveOld = await pendingAutosave();
    typeIntoEditor("b");

    resolveOld(new Response(null, { status: 204 }));
    await vi.advanceTimersByTimeAsync(0);

    expect(autosaved).not.toHaveBeenCalled();
  });

  it("エラー本文の読み込み中に新しい保存が終わっても、古い失敗を出さない", async () => {
    const text = Promise.withResolvers<string>();
    const response = new Response("", { status: 422 });
    vi.spyOn(response, "text").mockReturnValue(text.promise);
    await autosave(response);
    await autosave(new Response(null, { status: 204 }));

    text.resolve("古い本文が長すぎます");
    await vi.advanceTimersByTimeAsync(0);

    expect(errorText()).toBe("");
    expect(autosaved).toHaveBeenCalledTimes(1);
  });

  it("手動保存の失敗後に、古い自動保存の成功で表示を戻さない", async () => {
    const resolveOld = await pendingAutosave();
    typeIntoEditor("b");
    beginManualSave()(422, "新しい本文が長すぎます");

    resolveOld(new Response(null, { status: 204 }));
    await vi.advanceTimersByTimeAsync(0);

    expect(errorText()).toContain("下書きを保存できませんでした。");
    expect(textbox().getAttribute("aria-invalid")).toBe("true");
    expect(autosaved).not.toHaveBeenCalled();
  });

  it("自動保存の失敗後に、古い手動保存の成功による差し替えを止める", async () => {
    const finishOld = beginManualSave();
    await autosave(new Response("新しい本文が長すぎます", { status: 422 }));

    const event = finishOld(200, "old saved time");

    expect(event.defaultPrevented).toBe(true);
    expect(errorText()).toContain("自動保存できませんでした。");
    expect(textbox().getAttribute("aria-invalid")).toBe("true");
  });

  it("手動保存の成功後に、古い自動保存の失敗を表示しない", async () => {
    const resolveOld = await pendingAutosave();
    typeIntoEditor("b");
    beginManualSave()(204, "");

    resolveOld(new Response("古い本文が長すぎます", { status: 422 }));
    await vi.advanceTimersByTimeAsync(0);

    expect(errorText()).toBe("");
    expect(textbox().hasAttribute("aria-invalid")).toBe(false);
  });

  it("自動保存の成功後に、古い手動保存の失敗を表示しない", async () => {
    const finishOld = beginManualSave();
    await autosave(new Response(null, { status: 204 }));

    const event = finishOld(422, "古い本文が長すぎます");

    expect(event.defaultPrevented).toBe(true);
    expect(errorText()).toBe("");
    expect(textbox().hasAttribute("aria-invalid")).toBe(false);
  });

  it.each([422, 413])("自動保存の%dで本文をinvalidにし、修正後の保存成功で解除する", async (status) => {
    await autosave(new Response("本文が長すぎます", { status }));
    expect(textbox().getAttribute("aria-invalid")).toBe("true");
    expect(textbox().getAttribute("aria-describedby")).toBe("page-draft-save-error");

    await autosave(new Response(null, { status: 204 }));

    expect(textbox().hasAttribute("aria-invalid")).toBe(false);
    expect(textbox().getAttribute("aria-describedby")).toBe("page-draft-save-error");
  });

  it.each([422, 413])("手動保存の%dで本文をinvalidにし、修正後の保存成功で解除する", (status) => {
    beginManualSave()(status, "本文が長すぎます");
    expect(textbox().getAttribute("aria-invalid")).toBe("true");

    typeIntoEditor("a");
    beginManualSave()(204, "");

    expect(textbox().hasAttribute("aria-invalid")).toBe(false);
    expect(errorText()).toBe("");
  });

  it("公開エラーの本文を修正して自動保存できたら、古い文面とARIA状態を解除する", async () => {
    initializeWithPublishError();
    expect(textbox().getAttribute("aria-invalid")).toBe("true");

    await autosave(new Response(null, { status: 204 }));

    expect(document.getElementById("page-body-error")).toBeNull();
    expect(textbox().hasAttribute("aria-invalid")).toBe(false);
    expect(textbox().getAttribute("aria-describedby")).toBe("page-draft-save-error");
  });

  it("公開エラーの本文を修正して手動保存できたら、古い文面とARIA状態を解除する", () => {
    initializeWithPublishError();
    typeIntoEditor("a");

    beginManualSave()(204, "");

    expect(document.getElementById("page-body-error")).toBeNull();
    expect(textbox().hasAttribute("aria-invalid")).toBe(false);
    expect(textbox().getAttribute("aria-describedby")).toBe("page-draft-save-error");
  });
});

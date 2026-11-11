import { EditorView } from "codemirror";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { initializeDraftConflict } from "./draft-conflict";
import { initializeDraftVersion } from "./draft-version";
import { initializeEditors, stopAutosave } from "./markdown-editor";
import { initializePublishForms } from "./publish-handler";
import { beginHtmxDraftSave, initializeSaveStatus } from "./save-status";

interface MarkdownEditorContainer extends HTMLElement {
  _editorView?: EditorView;
}

beforeAll(() => initializeSaveStatus());

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

  it("ページがダークのときはエディタをダークのテーマにする", () => {
    document.documentElement.classList.add("dark");
    try {
      initializeEditors();
    } finally {
      document.documentElement.classList.remove("dark");
    }

    const container = document.querySelector<MarkdownEditorContainer>("[data-markdown-editor]");

    expect(container?._editorView?.state.facet(EditorView.darkTheme)).toBe(true);
  });

  it("ページがライトのときはエディタをライトのテーマにする", () => {
    initializeEditors();

    const container = document.querySelector<MarkdownEditorContainer>("[data-markdown-editor]");

    expect(container?._editorView?.state.facet(EditorView.darkTheme)).toBe(false);
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

  it("送信中の自動保存の後に待っている自動保存は、止めたら送らずに解決する", async () => {
    let resolveFirst: (response: Response) => void = () => {};
    fetchMock.mockImplementationOnce(() => new Promise<Response>((resolve) => (resolveFirst = resolve)));

    typeIntoEditor("a");
    await vi.advanceTimersByTimeAsync(500);
    typeIntoEditor("b");
    await vi.advanceTimersByTimeAsync(500);
    // 2件目は1件目の応答を待っている
    expect(fetchMock).toHaveBeenCalledTimes(1);

    let stopped = false;
    const stopping = stopAutosave().then(() => (stopped = true));
    await vi.advanceTimersByTimeAsync(0);
    expect(stopped).toBe(false);

    resolveFirst(new Response(null, { status: 200 }));
    await stopping;
    expect(stopped).toBe(true);
    expect(fetchMock).toHaveBeenCalledTimes(1);
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
    const ctx = { request: { body }, response: { status: 0 }, hx: {} as { redirect?: string }, text: "" };
    document
      .getElementById("save-draft")!
      .dispatchEvent(new CustomEvent("htmx:before:request", { bubbles: true, cancelable: true, detail: { ctx } }));
    // この節では順番待ちを通さず、draft-version.tsが送る直前に行う記録だけを写す
    beginHtmxDraftSave(ctx, document.querySelector<HTMLElement>("[data-markdown-editor]")!, body);
    // redirectは、応答のHX-Redirectとしてhtmxがctx.hxに入れる移動先
    return (status: number, text: string, redirect?: string): Event => {
      ctx.response.status = status;
      ctx.hx.redirect = redirect;
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

  it("本文を直した自動保存の送信中にタイトルだけを変えても、保存できたらエラーとARIA状態を戻す", async () => {
    await autosave(new Response("本文が長すぎます", { status: 422 }));
    const resolveFixed = await pendingAutosave();
    // タイトルの入力では自動保存が始まらないため、この応答のあとに表示を直す応答は来ない
    document.querySelector<HTMLInputElement>("#page_title")!.value = "New title";

    resolveFixed(new Response(null, { status: 204 }));
    await vi.advanceTimersByTimeAsync(0);

    expect(errorText()).toBe("");
    expect(textbox().hasAttribute("aria-invalid")).toBe(false);
    expect(autosaved).toHaveBeenCalledTimes(1);
  });

  it("入力の直後に押した手動保存は、同じ内容の自動保存があとから始まっても編集提案の作成画面へ移る", async () => {
    typeIntoEditor("a");
    const finishManual = beginManualSave();
    fetchMock.mockReturnValueOnce(new Promise<Response>(() => {}));
    await vi.advanceTimersByTimeAsync(1000);
    expect(fetchMock).toHaveBeenCalledTimes(1);

    const event = finishManual(204, "", "/s/test-space/topics/1/suggestions/new");

    expect(event.defaultPrevented).toBe(false);
  });

  it("手動保存の送信後にタイトルを変えたら、編集提案の作成画面へ移らない", () => {
    const finishManual = beginManualSave();
    document.querySelector<HTMLInputElement>("#page_title")!.value = "New title";

    const event = finishManual(204, "", "/s/test-space/topics/1/suggestions/new");

    expect(event.defaultPrevented).toBe(true);
    expect(errorText()).toBe("");
  });
});

// pages/page/edit.templの公開のフォームのうち、公開の前提にする版の項目と公開のボタンを写す
function publishFormMarkup(): string {
  return `
    <form id="publish-form" action="/s/test-space/pages/1" method="POST">
      <input type="hidden" name="draft_page_digest" value="v1" data-draft-page-digest-field>
      <button type="submit">Publish</button>
    </form>
  `;
}

// pages/page/edit.templの、下書きの競合の知らせを写す
function draftConflictMarkup(): string {
  return `
    <div data-draft-conflict data-reload-confirm="最新の内容を読み込みますか？" data-reload-url="/s/test-space/pages/1/edit" hidden>
      <div role="alert">
        <button type="button" data-draft-conflict-reload>最新の内容を読み込む</button>
        <button type="button" data-draft-conflict-overwrite>このまま上書きする</button>
      </div>
    </div>
  `;
}

describe("下書きの版の持ち回り", () => {
  let fetchMock: ReturnType<typeof vi.fn<typeof fetch>>;

  beforeAll(() => {
    initializeDraftVersion();
    initializeDraftConflict();
    initializePublishForms();
  });

  beforeEach(() => {
    vi.useFakeTimers();
    fetchMock = vi.fn<typeof fetch>();
    vi.stubGlobal("fetch", fetchMock);

    document.body.innerHTML = editorMarkup() + saveStatusMarkup() + draftConflictMarkup() + publishFormMarkup();
    const container = document.querySelector("[data-markdown-editor]")!;
    container.setAttribute("data-markdown-editor-draft-save-url", "/s/test-space/pages/1/draft_page");
    container.setAttribute("data-markdown-editor-draft-page-digest", "v1");
    initializeEditors();
  });

  afterEach(() => {
    const container = document.querySelector<MarkdownEditorContainer>("[data-markdown-editor]");
    container?._editorView?.destroy();
    document.body.innerHTML = "";
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  function typeIntoEditor(text: string): void {
    const view = document.querySelector<MarkdownEditorContainer>("[data-markdown-editor]")?._editorView;
    view?.dispatch({ changes: { from: 0, insert: text } });
  }

  function savedResponse(digest: string): Response {
    return new Response(null, { status: 204, headers: { ETag: `"${digest}"` } });
  }

  function conflictResponse(digest: string): Response {
    return new Response("下書きは、編集画面を開いたあとに別の場所で更新されています", {
      status: 409,
      headers: { ETag: `"${digest}"` },
    });
  }

  // sentDigestは、n番目 (0始まり) に送った保存が前提にした版を返す
  function sentDigest(n: number): string | null {
    const init = fetchMock.mock.calls[n]?.[1] as RequestInit | undefined;
    const body = init?.body as FormData | undefined;
    return (body?.get("pages_edit_form[draft_page_digest]") ?? body?.get("draft_page_digest")) as string | null;
  }

  async function autosave(): Promise<void> {
    typeIntoEditor("a");
    await vi.advanceTimersByTimeAsync(1000);
  }

  // htmxが下書き保存のボタンで送る要求を写す。htmxはhtmx:before:requestのあとにctx.fetchで送る
  async function manualSave(redirect?: string): Promise<Event> {
    const body = new FormData();
    body.set("title", document.querySelector<HTMLInputElement>("#page_title")!.value);
    body.set("body", document.querySelector<HTMLTextAreaElement>("#page_body")!.value);
    const request: RequestInit = { method: "PATCH", body };
    const ctx: {
      request: RequestInit;
      fetch: (input: string, init: RequestInit) => Promise<Response>;
      response?: { status: number };
      hx?: { redirect?: string };
      text?: string;
    } = {
      request,
      fetch: (input, init) => fetchMock(input, init),
    };
    document
      .getElementById("save-draft")!
      .dispatchEvent(new CustomEvent("htmx:before:request", { bubbles: true, cancelable: true, detail: { ctx } }));
    const response = await ctx.fetch("/s/test-space/pages/1/draft_page_revision", request);
    ctx.response = { status: response.status };
    ctx.hx = { redirect };
    ctx.text = await response.text();
    const event = new CustomEvent("htmx:after:request", { bubbles: true, cancelable: true, detail: { ctx } });
    document.getElementById("save-draft")!.dispatchEvent(event);
    return event;
  }

  function conflictShown(): boolean {
    return !document.querySelector<HTMLElement>("[data-draft-conflict]")!.hidden;
  }

  it("開いたときの版で保存し、成功したら応答の版を次の保存の前提にする", async () => {
    fetchMock.mockResolvedValueOnce(savedResponse("v2")).mockResolvedValueOnce(savedResponse("v3"));

    await autosave();
    await autosave();

    expect(sentDigest(0)).toBe("v1");
    expect(sentDigest(1)).toBe("v2");
  });

  it("送信中の自動保存があれば、その応答を待ってから次の自動保存を送る", async () => {
    const first = Promise.withResolvers<Response>();
    fetchMock.mockReturnValueOnce(first.promise).mockResolvedValueOnce(savedResponse("v3"));

    await autosave();
    await autosave();
    expect(fetchMock).toHaveBeenCalledTimes(1);

    first.resolve(savedResponse("v2"));
    await vi.advanceTimersByTimeAsync(0);

    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(sentDigest(1)).toBe("v2");
  });

  it("手動保存は送信中の自動保存の応答を待ち、その版を添えて送る", async () => {
    const first = Promise.withResolvers<Response>();
    fetchMock.mockReturnValueOnce(first.promise).mockResolvedValueOnce(savedResponse("v3"));

    await autosave();
    const saving = manualSave();
    await vi.advanceTimersByTimeAsync(0);
    expect(fetchMock).toHaveBeenCalledTimes(1);

    first.resolve(savedResponse("v2"));
    await saving;

    expect(sentDigest(1)).toBe("v2");
  });

  it("手動保存の応答の版を、次の自動保存の前提にする", async () => {
    fetchMock.mockResolvedValueOnce(savedResponse("v2")).mockResolvedValueOnce(savedResponse("v3"));

    await manualSave();
    await autosave();

    expect(sentDigest(0)).toBe("v1");
    expect(sentDigest(1)).toBe("v2");
  });

  it.each([204, 422])("待っていた手動保存は最新の入力を送り、%dの応答も送った内容で判定する", async (status) => {
    const first = Promise.withResolvers<Response>();
    fetchMock
      .mockReturnValueOnce(first.promise)
      .mockResolvedValueOnce(savedResponse("v3"))
      .mockResolvedValueOnce(status === 204 ? savedResponse("v4") : new Response("最新の本文が長すぎます", { status }));

    await autosave();
    // 2件目の自動保存と手動保存が、1件目の応答を待つ。
    await autosave();
    const saving = manualSave("/s/test-space/topics/1/suggestions/new");
    typeIntoEditor("最新の入力");
    document.querySelector<HTMLInputElement>("#page_title")!.value = "最新のタイトル";
    expect(fetchMock).toHaveBeenCalledTimes(1);

    // 次のdebounceが発火する前に順番待ちを進める。
    first.resolve(savedResponse("v2"));
    const event = await saving;

    expect(fetchMock).toHaveBeenCalledTimes(3);
    const autoBody = fetchMock.mock.calls[1]![1]!.body as FormData;
    const manualBody = fetchMock.mock.calls[2]![1]!.body as FormData;
    expect(autoBody.get("pages_edit_form[body]")).toBe("最新の入力aaPage body");
    expect(manualBody.get("body")).toBe(autoBody.get("pages_edit_form[body]"));
    expect(manualBody.get("title")).toBe("最新のタイトル");
    expect(sentDigest(2)).toBe("v3");
    expect(event.defaultPrevented).toBe(false);
    expect(document.querySelector("[data-draft-save-error]")!.textContent).toBe(
      status === 204 ? "" : "下書きを保存できませんでした。 最新の本文が長すぎます",
    );
    expect(document.querySelector('.cm-content[role="textbox"]')!.hasAttribute("aria-invalid")).toBe(status === 422);
  });

  it("競合したら知らせを出し、利用者が選ぶまで自動保存を送らない", async () => {
    fetchMock.mockResolvedValueOnce(conflictResponse("api"));

    await autosave();
    expect(conflictShown()).toBe(true);

    await autosave();
    expect(fetchMock).toHaveBeenCalledTimes(1);
    // 画面の入力は消さない
    expect(document.querySelector<HTMLTextAreaElement>("#page_body")!.value).toBe("aaPage body");
  });

  it("手動保存が競合しても、知らせを出して自動保存を止める", async () => {
    fetchMock.mockResolvedValueOnce(conflictResponse("api"));

    await manualSave();
    expect(conflictShown()).toBe(true);

    await autosave();
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("待っているあいだに前の保存が競合したら、待っていた自動保存は送らない", async () => {
    const first = Promise.withResolvers<Response>();
    fetchMock.mockReturnValueOnce(first.promise);

    await autosave();
    await autosave();
    first.resolve(conflictResponse("api"));
    await vi.advanceTimersByTimeAsync(0);

    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("「このまま上書きする」で、競合で受け取った版を前提に画面の内容を保存し、自動保存を再開する", async () => {
    fetchMock
      .mockResolvedValueOnce(conflictResponse("api"))
      .mockResolvedValueOnce(savedResponse("v2"))
      .mockResolvedValueOnce(savedResponse("v3"));
    await autosave();
    // happy-domは選択の変更を同期で通知し、CodeMirrorのフォーカスの処理と衝突するため、呼び出しだけを確かめる
    const view = document.querySelector<MarkdownEditorContainer>("[data-markdown-editor]")!._editorView!;
    const focus = vi.spyOn(view, "focus").mockImplementation(() => {});

    document.querySelector<HTMLButtonElement>("[data-draft-conflict-overwrite]")!.click();
    await vi.advanceTimersByTimeAsync(0);

    expect(conflictShown()).toBe(false);
    expect(sentDigest(1)).toBe("api");
    const body = (fetchMock.mock.calls[1]?.[1] as RequestInit).body as FormData;
    expect(body.get("pages_edit_form[body]")).toBe("aPage body");
    // 隠したボタンからエディタへフォーカスを戻す
    expect(focus).toHaveBeenCalled();

    await autosave();
    expect(sentDigest(2)).toBe("v2");
  });

  it("下書きが消えていて競合したら、下書きが無い前提で上書きする", async () => {
    fetchMock.mockResolvedValueOnce(conflictResponse("")).mockResolvedValueOnce(savedResponse("v2"));
    await autosave();

    document.querySelector<HTMLButtonElement>("[data-draft-conflict-overwrite]")!.click();
    await vi.advanceTimersByTimeAsync(0);

    expect(sentDigest(1)).toBe("");
  });

  it("「最新の内容を読み込む」は、確認を取り消したら何もしない", async () => {
    fetchMock.mockResolvedValueOnce(conflictResponse("api"));
    await autosave();
    const confirm = vi.fn(() => false);
    vi.stubGlobal("confirm", confirm);
    const replace = vi.fn();
    vi.stubGlobal("location", { href: "https://example.com/s/test-space/pages/1/edit", replace });

    document.querySelector<HTMLButtonElement>("[data-draft-conflict-reload]")!.click();

    expect(confirm).toHaveBeenCalledWith("最新の内容を読み込みますか？");
    expect(replace).not.toHaveBeenCalled();
    expect(conflictShown()).toBe(true);
  });

  it("「最新の内容を読み込む」を確認したら、編集画面を取り直す", async () => {
    fetchMock.mockResolvedValueOnce(conflictResponse("api"));
    await autosave();
    vi.stubGlobal(
      "confirm",
      vi.fn(() => true),
    );
    const replace = vi.fn();
    vi.stubGlobal("location", { href: "https://example.com/s/test-space/pages/1/edit#page_title", replace });

    document.querySelector<HTMLButtonElement>("[data-draft-conflict-reload]")!.click();

    expect(replace).toHaveBeenCalledWith("https://example.com/s/test-space/pages/1/edit");
  });

  function publishForm(): HTMLFormElement {
    return document.querySelector<HTMLFormElement>("#publish-form")!;
  }

  function publishDigest(): string {
    return document.querySelector<HTMLInputElement>("[data-draft-page-digest-field]")!.value;
  }

  // reopenWithPublishConflictは、公開が競合して再表示した編集画面を写す。知らせを表示した状態で描画し、
  // エディターは送信した版を前提にしたまま、今の版を競合の版として持つ
  function reopenWithPublishConflict(currentDigest: string): void {
    document.querySelector<MarkdownEditorContainer>("[data-markdown-editor]")!._editorView!.destroy();
    const container = document.querySelector("[data-markdown-editor]")!;
    container.setAttribute("data-markdown-editor-draft-conflict-digest", currentDigest);
    const conflict = document.querySelector<HTMLElement>("[data-draft-conflict]")!;
    conflict.setAttribute("data-draft-conflict-publish", "");
    conflict.hidden = false;
    initializeEditors();
  }

  it("保存が成功するたびに、公開のフォームの版を応答の版に書き換える", async () => {
    fetchMock.mockResolvedValueOnce(savedResponse("v2")).mockResolvedValueOnce(conflictResponse("api"));

    expect(publishDigest()).toBe("v1");
    await autosave();
    expect(publishDigest()).toBe("v2");

    // 競合した保存の版では書き換えない
    await autosave();
    expect(publishDigest()).toBe("v2");
  });

  it("公開は送信中の保存の応答を待ち、その版を前提にして送る", async () => {
    const first = Promise.withResolvers<Response>();
    fetchMock.mockReturnValueOnce(first.promise);
    const submit = vi.spyOn(publishForm(), "submit").mockImplementation(() => {});

    await autosave();
    const event = new SubmitEvent("submit", { bubbles: true, cancelable: true });
    publishForm().dispatchEvent(event);
    await vi.advanceTimersByTimeAsync(0);

    expect(event.defaultPrevented).toBe(true);
    expect(submit).not.toHaveBeenCalled();
    expect(publishForm().querySelector("button")!.disabled).toBe(true);

    first.resolve(savedResponse("v2"));
    await vi.waitFor(() => expect(submit).toHaveBeenCalledTimes(1));
    expect(publishDigest()).toBe("v2");

    // 公開の後に自動保存で下書きを作り直さない
    await autosave();
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("公開が競合して再表示した画面では、利用者が選ぶまで自動保存を送らない", async () => {
    reopenWithPublishConflict("api");

    await autosave();

    expect(fetchMock).not.toHaveBeenCalled();
    expect(publishDigest()).toBe("v1");
  });

  it("公開の競合で「このまま上書きする」を選ぶと、競合で受け取った版を前提に公開し直す", async () => {
    reopenWithPublishConflict("api");
    const submit = vi.spyOn(publishForm(), "submit").mockImplementation(() => {});

    document.querySelector<HTMLButtonElement>("[data-draft-conflict-overwrite]")!.click();

    await vi.waitFor(() => expect(submit).toHaveBeenCalledTimes(1));
    expect(publishDigest()).toBe("api");
    expect(conflictShown()).toBe(false);
    // 下書きの保存はしない
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("公開の競合で下書きが消えていたら、下書きが無い前提で公開し直す", async () => {
    reopenWithPublishConflict("");
    const submit = vi.spyOn(publishForm(), "submit").mockImplementation(() => {});

    document.querySelector<HTMLButtonElement>("[data-draft-conflict-overwrite]")!.click();

    await vi.waitFor(() => expect(submit).toHaveBeenCalledTimes(1));
    expect(publishDigest()).toBe("");
  });
});

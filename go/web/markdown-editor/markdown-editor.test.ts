import { EditorView } from "codemirror";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { initializeEditors, stopAutosave } from "./markdown-editor";

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
    <textarea id="page_body"></textarea>
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

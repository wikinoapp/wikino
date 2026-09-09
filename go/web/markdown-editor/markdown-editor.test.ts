import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { initializeEditors } from "./markdown-editor";

interface MarkdownEditorContainer extends HTMLElement {
  _editorView?: { destroy(): void };
}

// Mirror the body editor of pages/page/edit.templ: the visible label the editor is named from
// (no `for`, since CodeMirror's contenteditable cannot be a label's control), the container
// carrying the data-* config, and the hidden textarea that holds the value.
//
// [Ja] pages/page/edit.templ の本文エディタを写す。エディタの命名元になる可視ラベル (CodeMirror の
// contenteditable は label の関連先になれないため `for` は持たない)、data-* 設定を持つコンテナ、
// 値を保持する非表示の textarea。
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

  it("names the CodeMirror textbox with the visible body label", () => {
    initializeEditors();

    const textbox = document.querySelector<HTMLElement>('.cm-content[role="textbox"]');

    expect(textbox?.getAttribute("aria-labelledby")).toBe("page-body-label");
  });

  it("focuses the CodeMirror textbox when the visible body label is clicked", () => {
    initializeEditors();

    const label = document.getElementById("page-body-label") as HTMLLabelElement;
    const textbox = document.querySelector<HTMLElement>('.cm-content[role="textbox"]');

    label.click();

    expect(document.activeElement).toBe(textbox);
  });

  it("still initializes the editor when the label selector is absent", () => {
    const container = document.querySelector<MarkdownEditorContainer>("[data-markdown-editor]");
    container?.removeAttribute("data-markdown-editor-label");

    initializeEditors();

    const textbox = document.querySelector<HTMLElement>('.cm-content[role="textbox"]');

    expect(textbox).not.toBeNull();
    expect(textbox?.hasAttribute("aria-labelledby")).toBe(false);
  });

  it("still initializes the editor when the label selector matches nothing", () => {
    const container = document.querySelector<MarkdownEditorContainer>("[data-markdown-editor]");
    container?.setAttribute("data-markdown-editor-label", "#renamed-body-label");

    initializeEditors();

    const textbox = document.querySelector<HTMLElement>('.cm-content[role="textbox"]');

    expect(textbox).not.toBeNull();
    expect(textbox?.hasAttribute("aria-labelledby")).toBe(false);
  });

  it("leaves out aria-labelledby when the matched label has no id", () => {
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

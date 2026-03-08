import { type Page, type Locator } from "@playwright/test";

/**
 * CodeMirrorエディタの操作ヘルパー
 * Rails版の shared_helpers.rb と同等の機能を提供する
 */

const EDITOR_SELECTOR = ".cm-content";
const CONTAINER_SELECTOR = "[data-markdown-editor]";
const TEXTAREA_SELECTOR = '[data-markdown-editor-target="textarea"]';

function getEditor(page: Page): Locator {
  return page.locator(EDITOR_SELECTOR);
}

/** エディタにテキストを入力する */
export async function fillInEditor(page: Page, text: string): Promise<void> {
  const editor = getEditor(page);
  await editor.click();
  await editor.pressSequentially(text);
}

/** エディタのコンテンツを直接設定する（CodeMirror APIを使用） */
export async function setEditorContent(page: Page, text: string): Promise<void> {
  await page.evaluate(
    ({ content, selector }: { content: string; selector: string }) => {
      const container = document.querySelector(selector) as HTMLElement & {
        _editorView: {
          state: { doc: { length: number } };
          dispatch: (tr: unknown) => void;
        };
      };
      const view = container._editorView;
      view.dispatch({
        changes: { from: 0, to: view.state.doc.length, insert: content },
      });
    },
    { content: text, selector: CONTAINER_SELECTOR },
  );
}

/** エディタのコンテンツを取得する（hidden textareaから） */
export async function getEditorContent(page: Page): Promise<string> {
  return page.evaluate((selector: string) => {
    const textarea = document.querySelector(selector) as HTMLTextAreaElement;
    return textarea?.value || "";
  }, TEXTAREA_SELECTOR);
}

/** エディタの内容をすべてクリアする */
export async function clearEditor(page: Page): Promise<void> {
  const editor = getEditor(page);
  await editor.press("Control+a");
  await editor.press("Delete");
}

/** エディタ内でEnterキーを押す */
export async function pressEnterInEditor(page: Page): Promise<void> {
  const editor = getEditor(page);
  await editor.press("Enter");
}

/** エディタ内でTabキーを押す */
export async function pressTabInEditor(page: Page): Promise<void> {
  const editor = getEditor(page);
  await editor.press("Tab");
}

/** エディタ内でShift+Tabキーを押す */
export async function pressShiftTabInEditor(page: Page): Promise<void> {
  const editor = getEditor(page);
  await editor.press("Shift+Tab");
}

/** カーソルをドキュメントの先頭に移動する */
export async function moveCursorToStart(page: Page): Promise<void> {
  const editor = getEditor(page);
  await editor.press("Control+Home");
}

/** すべてのテキストを選択する */
export async function selectAllInEditor(page: Page): Promise<void> {
  const editor = getEditor(page);
  await editor.press("Control+a");
}

export interface CursorPosition {
  line: number;
  column: number;
  absolutePosition: number;
}

/** カーソル位置を取得する */
export async function getCursorPosition(page: Page): Promise<CursorPosition> {
  return page.evaluate((selector: string) => {
    const container = document.querySelector(selector) as HTMLElement & {
      _editorView: {
        state: {
          selection: { main: { head: number } };
          doc: { lineAt: (pos: number) => { number: number; from: number } };
        };
      };
    };
    const view = container._editorView;
    const cursor = view.state.selection.main.head;
    const line = view.state.doc.lineAt(cursor);

    return {
      line: line.number,
      column: cursor - line.from,
      absolutePosition: cursor,
    };
  }, CONTAINER_SELECTOR);
}

/** カーソル位置を設定する（line: 1-indexed, column: 0-indexed） */
export async function setCursorPosition(page: Page, line: number, column: number): Promise<void> {
  await page.evaluate(
    ({ line, column, selector }: { line: number; column: number; selector: string }) => {
      const container = document.querySelector(selector) as HTMLElement & {
        _editorView: {
          state: { doc: { line: (n: number) => { from: number } } };
          dispatch: (tr: unknown) => void;
          focus: () => void;
        };
      };
      const view = container._editorView;
      const lineInfo = view.state.doc.line(line);
      const position = lineInfo.from + column;

      view.dispatch({
        selection: { anchor: position, head: position },
      });
      view.focus();
    },
    { line, column, selector: CONTAINER_SELECTOR },
  );
}

/** 特定の行を改行文字を含めて選択する（line: 1-indexed） */
export async function selectLineWithNewline(page: Page, lineNumber: number): Promise<void> {
  await page.evaluate(
    ({ ln, selector }: { ln: number; selector: string }) => {
      const container = document.querySelector(selector) as HTMLElement & {
        _editorView: {
          state: { doc: { line: (n: number) => { from: number }; lines: number; length: number } };
          dispatch: (tr: unknown) => void;
          focus: () => void;
        };
      };
      const view = container._editorView;
      const doc = view.state.doc;
      const line = doc.line(ln);
      const nextLineStart = ln < doc.lines ? doc.line(ln + 1).from : doc.length;

      view.dispatch({
        selection: { anchor: line.from, head: nextLineStart },
      });
      view.focus();
    },
    { ln: lineNumber, selector: CONTAINER_SELECTOR },
  );
}

/** 複数行を改行文字を含めて選択する（startLine, endLine: 1-indexed） */
export async function selectMultipleLinesWithNewline(page: Page, startLine: number, endLine: number): Promise<void> {
  await page.evaluate(
    ({ start, end, selector }: { start: number; end: number; selector: string }) => {
      const container = document.querySelector(selector) as HTMLElement & {
        _editorView: {
          state: { doc: { line: (n: number) => { from: number }; lines: number; length: number } };
          dispatch: (tr: unknown) => void;
          focus: () => void;
        };
      };
      const view = container._editorView;
      const doc = view.state.doc;
      const startLineInfo = doc.line(start);
      const nextLineStart = end < doc.lines ? doc.line(end + 1).from : doc.length;

      view.dispatch({
        selection: { anchor: startLineInfo.from, head: nextLineStart },
      });
      view.focus();
    },
    { start: startLine, end: endLine, selector: CONTAINER_SELECTOR },
  );
}

export interface SelectionInfo {
  from: number;
  to: number;
  fromLine: number;
  toLine: number;
  fromColumn: number;
  toColumn: number;
  hasSelection: boolean;
}

/** 選択範囲の情報を取得する */
export async function getSelectionInfo(page: Page): Promise<SelectionInfo> {
  return page.evaluate((selector: string) => {
    const container = document.querySelector(selector) as HTMLElement & {
      _editorView: {
        state: {
          selection: { main: { from: number; to: number } };
          doc: { lineAt: (pos: number) => { number: number; from: number } };
        };
      };
    };
    const view = container._editorView;
    const selection = view.state.selection.main;
    const fromLine = view.state.doc.lineAt(selection.from);
    const toLine = view.state.doc.lineAt(selection.to);

    return {
      from: selection.from,
      to: selection.to,
      fromLine: fromLine.number,
      toLine: toLine.number,
      fromColumn: selection.from - fromLine.from,
      toColumn: selection.to - toLine.from,
      hasSelection: selection.from !== selection.to,
    };
  }, CONTAINER_SELECTOR);
}

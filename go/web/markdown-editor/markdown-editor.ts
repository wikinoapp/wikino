import { autocompletion, completionKeymap, closeBrackets, closeBracketsKeymap } from "@codemirror/autocomplete";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import {
  defaultHighlightStyle,
  syntaxHighlighting,
  indentOnInput,
  bracketMatching,
  foldGutter,
  foldKeymap,
} from "@codemirror/language";
import { searchKeymap, highlightSelectionMatches } from "@codemirror/search";
import { EditorState } from "@codemirror/state";
import {
  keymap,
  highlightSpecialChars,
  drawSelection,
  dropCursor,
  rectangularSelection,
  crosshairCursor,
  lineNumbers,
} from "@codemirror/view";
import { EditorView } from "codemirror";

import { fileDropHandler } from "./file-drop-handler";
import { FileUploadHandler } from "./file-upload-handler";
import { insertNewlineAndContinueList } from "./list-continuation";
import { clickManualSaveButton, handleManualSaveShortcut } from "./manual-save-handler";
import { pasteHandler } from "./paste-handler";
import { handleSubmitShortcut } from "./submit-handler";
import { handleTab, handleShiftTab } from "./tab-handler";
import { wikilinkCompletions } from "./wikilink-completions";

const AUTOSAVE_DEBOUNCE_MS = 500;

interface EditorConfig {
  container: HTMLElement;
  textarea: HTMLTextAreaElement;
  label: HTMLLabelElement | null;
  body: string;
  autofocus: boolean;
  draftSaveUrl: string;
  csrfToken: string;
  topicNumber: string;
  titleInput: HTMLInputElement;
  spaceIdentifier: string;
}

function createEditor(config: EditorConfig): EditorView {
  let debounceTimer: ReturnType<typeof setTimeout> | null = null;

  const state = EditorState.create({
    doc: config.body,
    extensions: [
      lineNumbers(),
      highlightSpecialChars(),
      history(),
      foldGutter(),
      drawSelection(),
      dropCursor(),
      EditorState.allowMultipleSelections.of(true),
      indentOnInput(),
      syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
      bracketMatching(),
      closeBrackets(),
      autocompletion({ override: [wikilinkCompletions(config.spaceIdentifier)] }),
      rectangularSelection(),
      crosshairCursor(),
      highlightSelectionMatches(),
      EditorView.lineWrapping,
      // ラベルの有無ではなくidの有無で判定する。aria-labelledbyはidを参照する属性なので、
      // id以外のセレクタで拾ったラベルだとエディタに空の参照が残ってしまう。
      ...(config.label?.id ? [EditorView.contentAttributes.of({ "aria-labelledby": config.label.id })] : []),
      keymap.of([
        { key: "Enter", run: insertNewlineAndContinueList },
        { key: "Tab", run: handleTab },
        { key: "Shift-Tab", run: handleShiftTab },
        { key: "Mod-Enter", run: handleSubmitShortcut },
        { key: "Mod-s", run: handleManualSaveShortcut },
        ...closeBracketsKeymap,
        ...defaultKeymap,
        ...searchKeymap,
        ...historyKeymap,
        ...foldKeymap,
        ...completionKeymap,
      ]),
      fileDropHandler,
      EditorView.domEventHandlers({
        paste: (event, view) => pasteHandler(view, event),
      }),
      EditorView.updateListener.of((update) => {
        if (update.docChanged) {
          config.textarea.value = update.state.doc.toString();
          config.textarea.dispatchEvent(new Event("input"));

          if (debounceTimer) {
            clearTimeout(debounceTimer);
          }
          debounceTimer = setTimeout(() => {
            saveAsDraft(config);
          }, AUTOSAVE_DEBOUNCE_MS);
        }
      }),
    ],
  });

  const view = new EditorView({
    state,
    parent: config.container,
  });

  if (config.autofocus) {
    view.focus();
  }

  return view;
}

async function saveAsDraft(config: EditorConfig): Promise<void> {
  if (!config.draftSaveUrl) return;

  const formData = new FormData();
  formData.append("pages_edit_form[topic_number]", config.topicNumber);
  formData.append("pages_edit_form[title]", config.titleInput.value);
  formData.append("pages_edit_form[body]", config.textarea.value);
  formData.append("csrf_token", config.csrfToken);

  try {
    const response = await fetch(config.draftSaveUrl, {
      method: "PATCH",
      body: formData,
    });

    if (response.ok) {
      window.dispatchEvent(new CustomEvent("draft-autosaved"));
    }
  } catch {
    // 自動保存の失敗は静かに無視する
  }
}

export function initializeEditors(): void {
  const containers = document.querySelectorAll<HTMLElement>("[data-markdown-editor]");

  containers.forEach((container) => {
    const textareaSelector = container.dataset.markdownEditorTextarea || "";
    const textarea = document.querySelector<HTMLTextAreaElement>(textareaSelector);
    if (!textarea) return;

    const titleSelector = container.dataset.markdownEditorTitle || "";
    const titleInput = document.querySelector<HTMLInputElement>(titleSelector);
    if (!titleInput) return;

    // ラベルはエディタのアクセシブルネームを供給するだけなので、セレクタが無い / 解決できない
    // ときは名前の無いエディタに退化させ、エディタ (とmain.jsがこの後に初期化するもの) までは
    // 巻き込まない。空文字列のセレクタではquerySelectorが例外を投げるため、検索結果ではなく
    // 文字列側でガードする。
    const labelSelector = container.dataset.markdownEditorLabel || "";
    const label = labelSelector ? document.querySelector<HTMLLabelElement>(labelSelector) : null;

    const body = container.dataset.markdownEditorBody || "";
    const autofocus = container.dataset.markdownEditorAutofocus === "true";
    const draftSaveUrl = container.dataset.markdownEditorDraftSaveUrl || "";
    const csrfToken = container.dataset.markdownEditorCsrfToken || "";
    const topicNumber = container.dataset.markdownEditorTopicNumber || "";
    const spaceIdentifier = container.dataset.markdownEditorSpaceIdentifier || "";

    const view = createEditor({
      container,
      textarea,
      label,
      body,
      autofocus,
      draftSaveUrl,
      csrfToken,
      topicNumber,
      titleInput,
      spaceIdentifier,
    });

    // 可視ラベルには関連付けられるフォーム部品が無い。入力面が `for` では指せない
    // CodeMirrorのcontenteditableであるため。ラベルが名前を与えている入力欄へフォーカスが移る
    // ようクリックを転送し、このフォームのネイティブ入力欄と同じ挙動にする。
    label?.addEventListener("click", () => {
      view.focus();
    });

    // タイトル入力欄でのMod-sでも手動保存を実行する。CodeMirrorのキーマップはエディタ
    // 本文しかカバーせず、これが無いとタイトルにフォーカス中はブラウザの「ページを保存」
    // ダイアログが出てしまう。
    titleInput.addEventListener("keydown", (event) => {
      // CodeMirrorのMod-sバインドと同じ完全一致にする。Shift / Alt併用時は無視し、
      // タイトルにフォーカス中もCtrl+Shift+Sなどのブラウザショートカットを妨げない。
      if ((event.metaKey || event.ctrlKey) && !event.shiftKey && !event.altKey && event.key.toLowerCase() === "s") {
        event.preventDefault();
        clickManualSaveButton();
      }
    });

    const uploadHandler = new FileUploadHandler(view, spaceIdentifier, csrfToken);

    view.dom.addEventListener("file-drop", ((e: CustomEvent) => {
      const { files, position } = e.detail as { files: File[]; position: number };
      uploadHandler.handleFileUpload(files, position);
    }) as EventListener);

    view.dom.addEventListener("media-paste", ((e: CustomEvent) => {
      const { file, position } = e.detail as { file: File; position: number };
      uploadHandler.handleFileUpload([file], position);
    }) as EventListener);

    view.dom.addEventListener("file-paste", ((e: CustomEvent) => {
      const { file, position } = e.detail as { file: File; position: number };
      uploadHandler.handleFileUpload([file], position);
    }) as EventListener);

    (container as HTMLElement & { _editorView: EditorView })._editorView = view;
  });
}

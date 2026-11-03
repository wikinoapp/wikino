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
import { Compartment, EditorState } from "@codemirror/state";
import {
  keymap,
  highlightSpecialChars,
  ViewPlugin,
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
import { beginDraftSave, clearSaveError, isCurrentDraftSave, showSaveError } from "./save-status";
import { handleSubmitShortcut } from "./submit-handler";
import { handleTab, handleShiftTab } from "./tab-handler";
import { wikilinkCompletions } from "./wikilink-completions";

const AUTOSAVE_DEBOUNCE_MS = 500;

// AutosaveStateはエディタ1つ分の自動保存の状態。下書きの削除前に自動保存を止め、送信中の
// 保存の完了を待てるようにするため、停止フラグと送信中の保存を保持する。
interface AutosaveState {
  debounceTimer: ReturnType<typeof setTimeout> | null;
  stopped: boolean;
  inFlight: Set<Promise<void>>;
}

const autosaveStates = new Set<AutosaveState>();

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
  describedBy: string;
  invalid: boolean;
}

function createEditor(config: EditorConfig): EditorView {
  const autosave: AutosaveState = { debounceTimer: null, stopped: false, inFlight: new Set() };
  autosaveStates.add(autosave);
  const validationAttributes = new Compartment();
  let describedBy = config.describedBy;
  let invalid = config.invalid;
  const bodyAttributes = () =>
    EditorView.contentAttributes.of({
      ...(describedBy ? { "aria-describedby": describedBy } : {}),
      ...(invalid ? { "aria-invalid": "true" } : {}),
    });

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
      // 本文のエラー (公開の失敗・保存の失敗) をエディタの説明として読み上げさせる。入力面が
      // CodeMirrorのcontenteditableで、エラーの要素をフォーム部品に関連付けられないため。
      validationAttributes.of(bodyAttributes()),
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
      // エディタを破棄したら自動保存の状態も手放し、以降のstopAutosaveが破棄済みのエディタの
      // 保存を待たないようにする
      ViewPlugin.define(() => ({
        destroy: () => {
          autosaveStates.delete(autosave);
          config.container.removeEventListener("draft-save-validation", handleValidationChange);
        },
      })),
      EditorView.domEventHandlers({
        paste: (event, view) => pasteHandler(view, event),
      }),
      EditorView.updateListener.of((update) => {
        if (update.docChanged) {
          config.textarea.value = update.state.doc.toString();
          config.textarea.dispatchEvent(new Event("input"));

          if (autosave.stopped) return;

          if (autosave.debounceTimer) {
            clearTimeout(autosave.debounceTimer);
          }
          autosave.debounceTimer = setTimeout(() => {
            autosave.debounceTimer = null;
            const saving = saveAsDraft(config);
            autosave.inFlight.add(saving);
            void saving.then(
              () => autosave.inFlight.delete(saving),
              () => autosave.inFlight.delete(saving),
            );
          }, AUTOSAVE_DEBOUNCE_MS);
        }
      }),
    ],
  });

  const view = new EditorView({
    state,
    parent: config.container,
  });

  // 公開時のエラーも、修正した本文を保存できたら解消する。属性を直接書き換えると
  // CodeMirrorの再描画で戻るため、Compartmentで入力面の属性を差し替える。
  function handleValidationChange(event: Event): void {
    const nextInvalid = (event as CustomEvent<{ invalid: boolean }>).detail.invalid;
    if (!nextInvalid) {
      document.getElementById("page-body-error")?.remove();
      describedBy = describedBy
        .split(/\s+/)
        .filter((id) => id !== "page-body-error")
        .join(" ");
    }
    if (invalid === nextInvalid && nextInvalid) return;
    invalid = nextInvalid;
    view.dispatch({ effects: validationAttributes.reconfigure(bodyAttributes()) });
  }
  config.container.addEventListener("draft-save-validation", handleValidationChange);

  if (config.autofocus) {
    view.focus();
  }

  return view;
}

// stopAutosaveはすべてのエディタの自動保存を止め、送信中の保存が終わったら解決するPromiseを
// 返す。保存の成否は問わない。下書きを削除する前に呼び、削除した下書きが自動保存で作り直され
// ないようにする。止めた自動保存は再開しない (削除後は画面遷移するため)。
export function stopAutosave(): Promise<void> {
  const pending: Promise<void>[] = [];
  autosaveStates.forEach((autosave) => {
    autosave.stopped = true;
    if (autosave.debounceTimer) {
      clearTimeout(autosave.debounceTimer);
      autosave.debounceTimer = null;
    }
    pending.push(...autosave.inFlight);
  });
  return Promise.allSettled(pending).then(() => undefined);
}

async function saveAsDraft(config: EditorConfig): Promise<void> {
  if (!config.draftSaveUrl) return;

  const formData = new FormData();
  formData.append("pages_edit_form[topic_number]", config.topicNumber);
  formData.append("pages_edit_form[title]", config.titleInput.value);
  formData.append("pages_edit_form[body]", config.textarea.value);
  formData.append("csrf_token", config.csrfToken);
  const request = beginDraftSave(config.container, formData);

  try {
    const response = await fetch(config.draftSaveUrl, {
      method: "PATCH",
      body: formData,
    });

    if (response.ok) {
      if (!isCurrentDraftSave(request)) return;
      clearSaveError();
      window.dispatchEvent(new CustomEvent("draft-autosaved"));
      return;
    }

    // 本文が上限を超えたときは、書き続けても保存されないことを知らせる
    const text = await response.text();
    if (isCurrentDraftSave(request)) showSaveError("autosave", response.status, text);
  } catch {
    // 通信の失敗は次の入力で保存し直せるため、表示を変えない
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
    const describedBy = container.dataset.markdownEditorDescribedby || "";
    const invalid = container.dataset.markdownEditorInvalid === "true";

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
      describedBy,
      invalid,
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

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
import { Controller } from "@hotwired/stimulus";
import { EditorView } from "codemirror";

import { wikilinkCompletions } from "../markdown-editor/wikilink-completions";
import { insertNewlineAndContinueList } from "../markdown-editor/list-continuation";
import { handleTab, handleShiftTab } from "../markdown-editor/tab-handler";
import { handleSubmitShortcut } from "../markdown-editor/submit-handler";
import { fileDropHandler, dropZoneStyles } from "../markdown-editor/file-drop-handler";
import { pasteHandler } from "../markdown-editor/paste-handler";
import { FileUploadHandler } from "../services/file-upload-handler";

export default class MarkdownEditorController extends Controller<HTMLDivElement> {
  static targets = ["codeMirror", "textarea"];
  static values = {
    spaceIdentifier: String,
    body: String,
    autofocus: Boolean,
  };

  declare readonly autofocusValue: boolean;
  declare readonly bodyValue: string;
  declare readonly spaceIdentifierValue: string;
  declare readonly codeMirrorTarget: HTMLDivElement;
  declare readonly textareaTarget: HTMLTextAreaElement;
  editorView!: EditorView;
  private fileUploadHandler!: FileUploadHandler;

  async connect() {
    await this.initializeEditor();
    this.setupFileHandlers();
  }

  async initializeEditor() {
    const state = EditorState.create({
      doc: this.bodyValue,
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
        autocompletion({ override: [await wikilinkCompletions(this.spaceIdentifierValue)] }),
        rectangularSelection(),
        crosshairCursor(),
        highlightSelectionMatches(),
        fileDropHandler,
        dropZoneStyles,
        EditorView.domEventHandlers({
          paste: (event, view) => pasteHandler(view, event as ClipboardEvent),
        }),
        keymap.of([
          { key: "Enter", run: insertNewlineAndContinueList },
          { key: "Tab", run: handleTab },
          { key: "Shift-Tab", run: handleShiftTab },
          { key: "Cmd-Enter", run: handleSubmitShortcut },
          { key: "Ctrl-Enter", run: handleSubmitShortcut },
          ...closeBracketsKeymap,
          ...defaultKeymap,
          ...searchKeymap,
          ...historyKeymap,
          ...foldKeymap,
          ...completionKeymap,
        ]),
        EditorView.updateListener.of((update: { docChanged: boolean; state: { doc: { toString(): string } } }) => {
          if (update.docChanged) {
            this.textareaTarget.value = update.state.doc.toString();
            this.textareaTarget.dispatchEvent(new Event("input"));
          }
        }),
      ],
    });

    this.editorView = new EditorView({
      state,
      parent: this.codeMirrorTarget,
    });

    if (this.autofocusValue) {
      this.editorView.focus();
    }

    // FileUploadHandlerのインスタンスを作成
    this.fileUploadHandler = new FileUploadHandler(this.editorView, this.spaceIdentifierValue);
  }

  setupFileHandlers() {
    // ファイルドロップイベントのハンドリング
    this.editorView.dom.addEventListener("file-drop", ((event: CustomEvent) => {
      const { files, position } = event.detail;
      this.fileUploadHandler.handleFileUpload(files, position);
    }) as EventListener);

    // 画像ペーストイベントのハンドリング
    this.editorView.dom.addEventListener("image-paste", ((event: CustomEvent) => {
      const { file, position } = event.detail;
      this.fileUploadHandler.handleFileUpload([file], position);
    }) as EventListener);
  }

  disconnect() {
    if (this.editorView) {
      this.editorView.destroy();
    }
  }
}

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
import { insertUploadPlaceholder, replacePlaceholderWithUrl, removePlaceholder } from "../markdown-editor/upload-placeholder";

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

  connect() {
    this.initializeEditor();
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
          paste: (event, view) => pasteHandler(view, event as ClipboardEvent)
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
  }

  setupFileHandlers() {
    // ファイルドロップイベントのハンドリング
    this.editorView.dom.addEventListener("file-drop", (event: CustomEvent) => {
      const { files, position } = event.detail;
      this.handleFileUpload(files, position);
    });

    // 画像ペーストイベントのハンドリング
    this.editorView.dom.addEventListener("image-paste", (event: CustomEvent) => {
      const { file, position } = event.detail;
      this.handleFileUpload([file], position);
    });
  }

  async handleFileUpload(files: File[], position: number) {
    for (const file of files) {
      // アップロード中のプレースホルダーを挿入
      const placeholderId = insertUploadPlaceholder(this.editorView, file.name, position);
      
      try {
        // ファイルアップロード処理（実装予定）
        // const url = await this.uploadFile(file);
        // replacePlaceholderWithUrl(this.editorView, placeholderId, url);
        
        // TODO: 実際のアップロード処理を実装
        console.log("File upload:", file.name);
        
        // 一時的にプレースホルダーを削除（アップロード機能実装時に削除）
        setTimeout(() => {
          removePlaceholder(this.editorView, placeholderId);
        }, 2000);
      } catch (error) {
        console.error("Upload error:", error);
        removePlaceholder(this.editorView, placeholderId);
        // エラー通知の表示（実装予定）
      }
    }
  }

  disconnect() {
    if (this.editorView) {
      this.editorView.destroy();
    }
  }
}

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
import {
  insertUploadPlaceholder,
  replacePlaceholderWithUrl,
  removePlaceholder,
} from "../markdown-editor/upload-placeholder";
import { DirectUpload, UploadError, UploadProgress } from "../services/direct-upload";
import { calculateFileChecksum } from "../utils/file-checksum";

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
  }

  setupFileHandlers() {
    // ファイルドロップイベントのハンドリング
    this.editorView.dom.addEventListener("file-drop", ((event: CustomEvent) => {
      const { files, position } = event.detail;
      this.handleFileUpload(files, position);
    }) as EventListener);

    // 画像ペーストイベントのハンドリング
    this.editorView.dom.addEventListener("image-paste", ((event: CustomEvent) => {
      const { file, position } = event.detail;
      this.handleFileUpload([file], position);
    }) as EventListener);
  }

  async handleFileUpload(files: File[], position: number) {
    for (const file of files) {
      // アップロード中のプレースホルダーを挿入
      const placeholderId = insertUploadPlaceholder(this.editorView, file.name, position);

      try {
        // ファイルのMD5チェックサムを計算
        const checksum = await calculateFileChecksum(file);

        // プリサイン用URLを取得
        const presignResponse = await fetch(`/s/${this.spaceIdentifierValue}/attachments/presign`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "X-CSRF-Token": document.querySelector('meta[name="csrf-token"]')?.getAttribute("content") || "",
          },
          body: JSON.stringify({
            filename: file.name,
            content_type: file.type,
            byte_size: file.size,
            checksum: checksum,
          }),
        });

        if (!presignResponse.ok) {
          throw new UploadError("プリサイン用URLの取得に失敗しました");
        }

        const { directUploadUrl, directUploadHeaders, blobSignedId } = await presignResponse.json();

        // DirectUploadを使用してファイルをアップロード
        const uploader = new DirectUpload(
          file,
          directUploadUrl,
          directUploadHeaders,
          (progress: UploadProgress) => {
            // 進捗状況のログ（必要に応じてUIに反映）
            console.log(`Uploading ${file.name}: ${progress.percentage}%`);
          }
        );

        await uploader.upload();

        // アップロード成功後、Active StorageのURLを生成
        const attachmentUrl = `/rails/active_storage/blobs/redirect/${blobSignedId}/${encodeURIComponent(file.name)}`;

        // プレースホルダーをURLに置換
        replacePlaceholderWithUrl(this.editorView, placeholderId, attachmentUrl, file.name);
      } catch (error) {
        console.error("Upload error:", error);
        removePlaceholder(this.editorView, placeholderId);

        // エラーメッセージの表示
        if (error instanceof UploadError) {
          this.showErrorMessage(error.message);
        } else {
          this.showErrorMessage("ファイルのアップロードに失敗しました");
        }
      }
    }
  }

  private showErrorMessage(message: string) {
    // エラーメッセージを表示（一時的な実装）
    // TODO: 適切なエラー通知システムに置き換える
    const notification = document.createElement("div");
    notification.textContent = message;
    notification.style.cssText = `
      position: fixed;
      top: 20px;
      right: 20px;
      background: #ef4444;
      color: white;
      padding: 12px 24px;
      border-radius: 6px;
      box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
      z-index: 10000;
    `;

    document.body.appendChild(notification);

    setTimeout(() => {
      notification.remove();
    }, 5000);
  }

  disconnect() {
    if (this.editorView) {
      this.editorView.destroy();
    }
  }
}

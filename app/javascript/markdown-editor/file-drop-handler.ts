import { EditorView, ViewPlugin, ViewUpdate } from "@codemirror/view";
import { EditorState, Transaction } from "@codemirror/state";

// ファイルドロップハンドラー
export const fileDropHandler = ViewPlugin.fromClass(
  class {
    private view: EditorView;
    private dragCounter = 0;
    private dropZone: HTMLDivElement | null = null;

    constructor(view: EditorView) {
      this.view = view;
      this.setupEventListeners();
    }

    setupEventListeners() {
      const dom = this.view.dom;

      // ドラッグイベントをバインド
      dom.addEventListener("dragenter", this.handleDragEnter.bind(this));
      dom.addEventListener("dragleave", this.handleDragLeave.bind(this));
      dom.addEventListener("dragover", this.handleDragOver.bind(this));
      dom.addEventListener("drop", this.handleDrop.bind(this));
    }

    handleDragEnter(e: DragEvent) {
      e.preventDefault();
      e.stopPropagation();

      this.dragCounter++;

      // ファイルがドラッグされているか確認
      if (e.dataTransfer?.types.includes("Files")) {
        this.showDropZone();
      }
    }

    handleDragLeave(e: DragEvent) {
      e.preventDefault();
      e.stopPropagation();

      this.dragCounter--;

      if (this.dragCounter === 0) {
        this.hideDropZone();
      }
    }

    handleDragOver(e: DragEvent) {
      e.preventDefault();
      e.stopPropagation();

      // ドロップ効果を設定
      if (e.dataTransfer) {
        e.dataTransfer.dropEffect = "copy";
      }
    }

    handleDrop(e: DragEvent) {
      e.preventDefault();
      e.stopPropagation();

      this.dragCounter = 0;
      this.hideDropZone();

      const files = e.dataTransfer?.files;
      if (!files || files.length === 0) return;

      // ドロップ位置のカーソル位置を取得
      const pos = this.view.posAtCoords({ x: e.clientX, y: e.clientY });

      // ファイルドロップイベントを発火
      this.view.dom.dispatchEvent(
        new CustomEvent("file-drop", {
          detail: {
            files: Array.from(files),
            position: pos || this.view.state.selection.main.head,
          },
          bubbles: true,
        }),
      );
    }

    showDropZone() {
      if (this.dropZone) return;

      this.dropZone = document.createElement("div");
      this.dropZone.className =
        "cm-drop-zone absolute inset-1 border-2 border-dashed " +
        "border-gray-500/50 rounded-md bg-transparent pointer-events-none";
      this.dropZone.style.zIndex = "1000";

      this.view.dom.style.position = "relative";
      this.view.dom.appendChild(this.dropZone);
    }

    hideDropZone() {
      if (this.dropZone) {
        this.dropZone.remove();
        this.dropZone = null;
      }
    }

    destroy() {
      this.hideDropZone();
    }

    update(update: ViewUpdate) {
      // 必要に応じて更新処理
    }
  },
);

// ドロップゾーンのスタイル（現在は使用していない）
export const dropZoneStyles = EditorView.theme({});

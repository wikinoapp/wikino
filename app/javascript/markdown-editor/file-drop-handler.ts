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
            position: pos || this.view.state.selection.main.head
          },
          bubbles: true
        })
      );
    }

    showDropZone() {
      if (this.dropZone) return;
      
      this.dropZone = document.createElement("div");
      this.dropZone.className = "cm-drop-zone";
      this.dropZone.innerHTML = `
        <div class="cm-drop-zone-content">
          <svg class="cm-drop-zone-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
            <polyline points="7 10 12 15 17 10" />
            <line x1="12" y1="15" x2="12" y2="3" />
          </svg>
          <p class="cm-drop-zone-text">ファイルをドロップしてアップロード</p>
        </div>
      `;
      
      // スタイルを適用
      Object.assign(this.dropZone.style, {
        position: "absolute",
        top: "0",
        left: "0",
        right: "0",
        bottom: "0",
        backgroundColor: "rgba(0, 0, 0, 0.1)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        zIndex: "1000",
        pointerEvents: "none"
      });
      
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
  }
);

// ドロップゾーンのスタイル
export const dropZoneStyles = EditorView.theme({
  ".cm-drop-zone-content": {
    backgroundColor: "white",
    borderRadius: "8px",
    padding: "32px",
    boxShadow: "0 4px 6px rgba(0, 0, 0, 0.1)",
    textAlign: "center"
  },
  ".cm-drop-zone-icon": {
    width: "48px",
    height: "48px",
    marginBottom: "16px",
    color: "#6b7280"
  },
  ".cm-drop-zone-text": {
    fontSize: "16px",
    color: "#4b5563",
    margin: "0"
  }
});
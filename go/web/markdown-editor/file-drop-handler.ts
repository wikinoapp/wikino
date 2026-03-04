import { EditorView, ViewPlugin, ViewUpdate } from "@codemirror/view";

export const fileDropHandler = ViewPlugin.fromClass(
  class {
    private view: EditorView;
    private dragCounter = 0;
    private dropZone: HTMLDivElement | null = null;
    private boundHandlers: {
      dragenter: (e: DragEvent) => void;
      dragleave: (e: DragEvent) => void;
      dragover: (e: DragEvent) => void;
      drop: (e: DragEvent) => void;
    };

    constructor(view: EditorView) {
      this.view = view;
      this.boundHandlers = {
        dragenter: this.handleDragEnter.bind(this),
        dragleave: this.handleDragLeave.bind(this),
        dragover: this.handleDragOver.bind(this),
        drop: this.handleDrop.bind(this),
      };
      this.setupEventListeners();
    }

    setupEventListeners() {
      const dom = this.view.dom;

      dom.addEventListener("dragenter", this.boundHandlers.dragenter);
      dom.addEventListener("dragleave", this.boundHandlers.dragleave);
      dom.addEventListener("dragover", this.boundHandlers.dragover);
      dom.addEventListener("drop", this.boundHandlers.drop);
    }

    handleDragEnter(e: DragEvent) {
      e.preventDefault();
      e.stopPropagation();

      this.dragCounter++;

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

      const pos = this.view.posAtCoords({ x: e.clientX, y: e.clientY });

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
      const dom = this.view.dom;
      dom.removeEventListener("dragenter", this.boundHandlers.dragenter);
      dom.removeEventListener("dragleave", this.boundHandlers.dragleave);
      dom.removeEventListener("dragover", this.boundHandlers.dragover);
      dom.removeEventListener("drop", this.boundHandlers.drop);
    }

    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    update(_update: ViewUpdate) {}
  },
);

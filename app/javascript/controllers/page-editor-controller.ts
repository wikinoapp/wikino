import { EditorState } from '@codemirror/state';
import { Controller } from '@hotwired/stimulus';
import { EditorView, basicSetup } from 'codemirror';

export default class extends Controller<HTMLDivElement> {
  static targets = ['codeMirror', 'textarea'];
  static values = {
    body: String,
  };

  declare readonly bodyValue: string;
  declare readonly codeMirrorTarget: HTMLDivElement;
  declare readonly textareaTarget: HTMLTextAreaElement;
  editorView: EditorView;

  connect() {
    this.initializeEditor();
  }

  initializeEditor() {
    const state = EditorState.create({
      doc: this.bodyValue,
      extensions: [
        basicSetup,
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            this.textareaTarget.value = update.state.doc.toString();
            this.textareaTarget.dispatchEvent(new Event('input'));
          }
        }),
      ],
    });

    this.editorView = new EditorView({
      state,
      parent: this.codeMirrorTarget,
    });
  }

  disconnect() {
    if (this.editorView) {
      this.editorView.destroy();
    }
  }
}

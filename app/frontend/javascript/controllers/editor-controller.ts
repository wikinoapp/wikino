// import { EditorState, EditorView, basicSetup } from '@codemirror/next/basic-setup';
import { closeBrackets, closeBracketsKeymap } from '@codemirror/next/closebrackets';
import { history, historyKeymap } from '@codemirror/next/history';
import { EditorState } from '@codemirror/next/state';
import { EditorView, keymap } from '@codemirror/next/view';
import { Controller } from 'stimulus';

import { nonotoKeymap } from '../codemirror/commands';
import { listComplement } from '../codemirror/extensions/list-complement';

export default class extends Controller {
  element!: HTMLElement;
  noteDatabaseId!: string;
  noteBody!: string;

  connect() {
    this.noteDatabaseId = this.data.get('noteDatabaseId') as string;
    this.noteBody = this.data.get('noteBody') as string;
    console.log('this.noteBody: ', this.noteBody);

    const editorView = new EditorView({
      // state: EditorState.create({ doc: this.noteBody, extensions: [this.eventHandlers()] }),
      state: EditorState.create({
        doc: this.noteBody,
        extensions: [
          closeBrackets(),
          history(),
          // listComplement(),
          keymap([...closeBracketsKeymap, ...historyKeymap, ...nonotoKeymap]),
        ],
      }),
    });

    this.element.appendChild(editorView.dom);
  }

  // eventHandlers() {
  //   return EditorView.domEventHandlers({
  //     keydown(event, view) {
  //       console.log('view: ', view.state.selection.primary);
  //       let transaction = view.state.update({ changes: { from: 0, insert: '0' } });
  //       view.dispatch(transaction);
  //       return false;
  //     },
  //   });
  // }
}

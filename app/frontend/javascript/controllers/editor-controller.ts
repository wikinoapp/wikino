import { closeBrackets, closeBracketsKeymap } from '@codemirror/next/closebrackets';
import { history, historyKeymap } from '@codemirror/next/history';
import { EditorState } from '@codemirror/next/state';
import { EditorView, keymap } from '@codemirror/next/view';
import { Controller } from 'stimulus';

import { nonotoKeymap } from '../codemirror/commands';

export default class extends Controller {
  element!: HTMLElement;
  noteDatabaseId!: string;
  noteBody!: string;

  connect() {
    this.noteDatabaseId = this.data.get('noteDatabaseId') as string;
    this.noteBody = this.data.get('noteBody') as string;
    console.log('this.noteBody: ', this.noteBody);

    const editorView = new EditorView({
      state: EditorState.create({
        doc: this.noteBody,
        extensions: [closeBrackets(), history(), keymap([...closeBracketsKeymap, ...historyKeymap, ...nonotoKeymap])],
      }),
    });

    this.element.appendChild(editorView.dom);
  }
}

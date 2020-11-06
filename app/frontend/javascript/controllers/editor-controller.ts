import { autocompletion } from '@codemirror/next/autocomplete';
import { closeBrackets, closeBracketsKeymap } from '@codemirror/next/closebrackets';
import { history, historyKeymap } from '@codemirror/next/history';
import { EditorState } from '@codemirror/next/state';
import { EditorView, keymap } from '@codemirror/next/view';
import { Controller } from 'stimulus';

import { nonotoKeymap } from '../codemirror/commands';
import { autoSave } from '../codemirror/extensions/auto-save';
import { autocompletionPlugin } from '../codemirror/extensions/autocomplete';
import { linkNoteCompletionSource } from '../codemirror/extensions/link-note';

export default class extends Controller {
  element!: HTMLElement;
  noteDatabaseId!: string;
  noteBody!: string;

  connect() {
    this.noteDatabaseId = this.data.get('noteDatabaseId') as string;
    this.noteBody = this.data.get('noteBody') as string;

    const editorView = new EditorView({
      state: EditorState.create({
        doc: this.noteBody,
        extensions: [
          autoSave(this.noteDatabaseId),
          autocompletion({
            override: [linkNoteCompletionSource]
          }),
          autocompletionPlugin,
          closeBrackets(),
          history(),
          keymap([...closeBracketsKeymap, ...historyKeymap, ...nonotoKeymap]),
        ],
      }),
    });

    this.element.appendChild(editorView.dom);
  }
}

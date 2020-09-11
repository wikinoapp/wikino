import { autocompletion, startCompletion, closeCompletion, CompletionSource } from '@codemirror/next/autocomplete';
import { closeBrackets, closeBracketsKeymap } from '@codemirror/next/closebrackets';
import { basicSetup } from '@codemirror/next/basic-setup';
import { history, historyKeymap } from '@codemirror/next/history';
import { EditorState } from '@codemirror/next/state';
import { EditorView, keymap } from '@codemirror/next/view';
import { Controller } from 'stimulus';

import { nonotoKeymap } from '../codemirror/commands';
import { autoSave } from '../codemirror/extensions/auto-save';
import { linkNote, linkNoteCompletionSource } from '../codemirror/extensions/link-note';

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
          // linkNote(),
          autocompletion({
            override: [linkNoteCompletionSource],
          }),
          closeBrackets(),
          history(),
          keymap([...closeBracketsKeymap, ...historyKeymap, ...nonotoKeymap]),
        ],
      }),
    });

    this.element.appendChild(editorView.dom);
  }
}

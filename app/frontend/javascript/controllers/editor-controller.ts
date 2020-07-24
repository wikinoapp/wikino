import CodeMirror, { Editor, EditorFromTextArea } from 'codemirror';
import gfm from 'codemirror/mode/gfm/gfm.js';
import { Controller } from 'stimulus';

export default class extends Controller {
  static targets = ['textArea'];

  textAreaTarget!: HTMLTextAreaElement;

  initialize() {
    const editor: EditorFromTextArea = CodeMirror.fromTextArea(this.textAreaTarget, {
      mode: {
        name: 'gfm',
        gitHubSpice: true,
      },
      lineNumbers: true,
      indentWithTabs: false,
      tabSize: 2,
    });

    editor.setOption('extraKeys', {
      Tab: (cm: Editor) => {
        if (cm.somethingSelected()) {
          cm.indentSelection('add');
          return;
        }

        cm.execCommand('insertSoftTab');
      },

      'Shift-Tab': (cm: Editor) => {
        cm.indentSelection('subtract');
      },

      Enter: (cm: Editor) => {
        const doc = cm.getDoc();
        const cursor = doc.getCursor();
        const prevLine = doc.getLine(cursor.line);

        const trimmedPrevLine = prevLine.trim();
        const indentSpaceMatches = prevLine.match(/^ +/);
        const indentSpace = indentSpaceMatches ? indentSpaceMatches[0] : '';
        const firstChar = trimmedPrevLine.charAt(0);

        let replacement = '';
        if (firstChar === '-' || firstChar === '*') {
          if (trimmedPrevLine === '-' || trimmedPrevLine === '*') {
            doc.getLineHandle(cursor.line).text = '';
            replacement = `\n${indentSpace}`;
          } else {
            replacement = `\n${indentSpace}${firstChar} `;
          }
        } else {
          replacement = `\n${indentSpace}`;
        }

        cm.replaceSelection(replacement);
      },
    });
  }
}

import CodeMirror, { Editor, EditorFromTextArea } from 'codemirror';
import 'codemirror/mode/gfm/gfm';

import { Controller } from 'stimulus';

export default class extends Controller {
  static targets = ['textArea'];

  textAreaTarget!: HTMLTextAreaElement;

  initialize() {
    const editor: EditorFromTextArea = CodeMirror.fromTextArea(this.textAreaTarget, {
      mode: {
        name: 'gfm',
        highlightFormatting: true,
        gitHubSpice: false,
      },
      theme: 'default',
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

    editor.on('inputRead', (cm: Editor, changeObj: any) => {
      // console.log('event.key: ', event.key);
      // console.log('changeObj: ', changeObj);
      const doc = cm.getDoc();
      const cursor = doc.getCursor();
      // console.log('cursor: ', cursor);
      const currentLine = doc.getLine(cursor.line);
      // console.log('currentLine: ', currentLine);
      const prevChars = currentLine.slice(0, cursor.ch);
      console.log('prevChars: ', prevChars);
      // console.log('changed! currentLine: ', currentLine);
      // console.log('getSelection: ', doc.getSelection());

      const symbolValue = this.symbolValue(prevChars, '[');
      console.log('symbolValue: ', symbolValue);

      [
        ['(', ')'],
        ['[', ']'],
        ['{', '}'],
      ].forEach((pair) => {
        if (changeObj.text[0] === pair[0]) {
          cm.replaceSelection(pair[1], 'start');
        }
      });

      const isSurrounded = this.isSurrounded(currentLine, cursor.ch, '[', ']');
      // console.log('isSurrounded: ', isSurrounded);
      if (isSurrounded) {
        const surroundedChars = this.surroundedChars(currentLine, cursor.ch, '[', ']').replace(/^\[(.*)\]$/, '$1');
        // console.log('surroundedChars: ', surroundedChars);
      }
    });
  }

  symbolValue(str: string, symbol: string) {
    const startCharMatches = str
      .split('')
      .reverse()
      .join('')
      .match(new RegExp(`^\\S+?\\${symbol}`));

    return startCharMatches
      ? startCharMatches[0]
          .split('')
          .reverse()
          .join('')
          .replace(new RegExp(`\\${symbol}`), '')
      : '';
  }

  startChars(line: string, ch: number, startChar: string) {
    const prevChars = line.slice(0, ch);
    // console.log('prevChars: ', prevChars);
    const prevMatches = prevChars
      .split('')
      .reverse()
      .join('')
      .match(new RegExp(`^(.*?)\\${startChar}`));

    return prevMatches ? prevMatches[0].split('').reverse().join('') : '';
  }

  endChars(line: string, ch: number, endChar: string) {
    const nextChars = line.slice(ch, line.length);
    // console.log('nextChars: ', nextChars);
    const nextMatches = nextChars.match(new RegExp(`^(.*?)\\${endChar}`));

    return nextMatches ? nextMatches[0] : '';
  }

  isSurrounded(line: string, ch: number, startChar: string, endChar: string) {
    // console.log(`line: ${line}, ch: ${ch}`);
    // console.log('startFrom: ', startFrom);

    // console.log('endTo: ', endTo);
    const startChars = this.startChars(line, ch, startChar);
    const endChars = this.endChars(line, ch, endChar);

    return startChars.includes(startChar) && endChars.includes(endChar);
  }

  surroundedChars(line: string, ch: number, startChar: string, endChar: string) {
    const startChars = this.startChars(line, ch, startChar);
    const endChars = this.endChars(line, ch, endChar);
    // console.log(`startChars: ${startChars}, endChars: ${endChars}`);

    return startChars + endChars;
  }
}

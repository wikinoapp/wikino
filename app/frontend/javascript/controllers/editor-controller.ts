import axios from 'axios';
import CodeMirror, { Doc, Editor, EditorFromTextArea, Position } from 'codemirror';
import { Controller } from 'stimulus';

import 'codemirror/mode/gfm/gfm';

import { EventDispatcher } from '../utils/event-dispatcher';

export default class extends Controller {
  static targets = ['hints', 'textArea'];

  hintsTarget!: HTMLElement;
  textAreaTarget!: HTMLTextAreaElement;
  isHintsDisplayed!: boolean;

  initialize() {
    let cursor: Position;
    let currentLine: string;
    let doc: Doc;

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
        doc = cm.getDoc();
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
      doc = cm.getDoc();
      cursor = doc.getCursor();
      // console.log('cursor: ', cursor);
      const inputChar = changeObj.text[0];
      // console.log('inputChar: ', inputChar);
      currentLine = doc.getLine(cursor.line);
      // console.log('currentLine: ', currentLine);
      const prevChars = currentLine.slice(0, cursor.ch);
      // console.log('prevChars: ', prevChars);
      // console.log('changed! currentLine: ', currentLine);
      // console.log('getSelection: ', doc.getSelection());

      // let bracketStartPos: Position;
      // if (inputChar === '[') {
      //   bracketStartPos = changeObj.from;
      // }

      [
        ['(', ')'],
        ['[', ']'],
        ['{', '}'],
      ].forEach((pair) => {
        if (inputChar === pair[0]) {
          cm.replaceSelection(pair[1], 'start');
        }
      });

      const linkTitle = this.symbolValue(prevChars, '[');
      console.log('linkTitle: ', linkTitle);
      if (linkTitle) {
        // const cursorElm: HTMLElement | null = this.element.querySelector('.CodeMirror-cursor');
        // console.log('cursorElm: ', cursorElm);

        axios
          .get('/api/internal/notes', {
            params: {
              q: linkTitle,
            },
          })
          .then((res) => {
            const hintsHtml = res.data ? res.data : null;
            // console.log('hintsHtml: ', hintsHtml);
            this.hintsTarget.innerHTML = hintsHtml;
            this.isHintsDisplayed = true;
            new EventDispatcher('editor-hints:show').dispatch();

            const cursorCoords = cm.cursorCoords(false, 'window');
            // console.log('cursorCoords: ', cursorCoords);
            this.hintsTarget.style.left = `${cursorCoords.left - 10}px`;
            this.hintsTarget.style.top = `${cursorCoords.top + 25}px`;
            // const bracketStartCoords = cm.charCoords(bracketStartPos);
            // console.log('bracketStartCoords: ', bracketStartCoords);
          })
          .catch((err) => {
            console.log('err: ', err);
          });
      }
    });

    editor.on('keydown', (cm: Editor, event: KeyboardEvent) => {
      // console.log('keydown! event.code: ', event.code);
      doc = cm.getDoc();
      cursor = doc.getCursor();

      if (this.isHintsDisplayed && this.isKeyForHints(event.code)) {
        new EventDispatcher('editor-hints:keydown', { code: event.code }).dispatch();
        event.preventDefault();
      }
    });

    document.addEventListener('editor:select-hint', (event: any) => {
      const { selectedNoteId, selectedNoteTitle } = event.detail;
      // console.log('selectedNoteId: ', selectedNoteId);
      console.log('currentLine: ', currentLine);
      console.log('cursor.ch: ', cursor.ch);
      const prevChars = currentLine.slice(0, cursor.ch);
      // console.log('prevChars: ', prevChars);
      const startBracketMatch = this.reverseString(prevChars).match(/\[/);
      console.log('startBracketMatch: ', startBracketMatch);
      let startBracketIndex: number = -1;
      if (typeof startBracketMatch?.index === 'number') {
        startBracketIndex = cursor.ch - (startBracketMatch.index + 1);
        console.log('startBracketIndex: ', startBracketIndex);
      }

      const afterChars = currentLine.slice(cursor.ch);
      console.log('afterChars: ', afterChars);
      const endBracketMatch = afterChars.match(/]/);
      console.log('endBracketMatch: ', endBracketMatch);
      let endBracketIndex: number = -1;
      if (typeof endBracketMatch?.index === 'number') {
        endBracketIndex = cursor.ch + endBracketMatch.index;
        console.log('endBracketIndex: ', endBracketIndex);
      }

      if (startBracketIndex >= 0 && endBracketIndex >= 0) {
        doc.replaceRange(
          `[${selectedNoteTitle}](/notes/${selectedNoteId})`,
          { line: cursor.line, ch: startBracketIndex },
          { line: cursor.line, ch: endBracketIndex + 1 },
        );

        this.isHintsDisplayed = false;
        new EventDispatcher('editor-hints:hide').dispatch();
      }
    });
  }

  isKeyForHints(code: string) {
    return ['ArrowUp', 'ArrowDown', 'Enter', 'Escape'].includes(code);
  }

  symbolValue(str: string, symbol: string) {
    const startCharMatches = this.reverseString(str).match(new RegExp(`^\\S+?\\${symbol}`));

    return startCharMatches ? this.reverseString(startCharMatches[0]).replace(new RegExp(`\\${symbol}`), '') : null;
  }

  reverseString(str: string) {
    return str.split('').reverse().join('');
  }
}

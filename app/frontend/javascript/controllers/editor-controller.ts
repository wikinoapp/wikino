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
  isComposing!: boolean;
  cm!: Editor;
  pos!: Position;
  doc!: Doc;

  initialize() {
    this.cm = CodeMirror.fromTextArea(this.textAreaTarget, {
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
    this.doc = this.cm.getDoc();
    this.pos = this.doc.getCursor();

    this.cm.setOption('extraKeys', {
      Tab: () => {
        if (this.cm.somethingSelected()) {
          this.cm.indentSelection('add');
          return;
        }

        this.cm.execCommand('insertSoftTab');
      },

      'Shift-Tab': () => {
        this.cm.indentSelection('subtract');
      },

      Enter: () => {
        console.log('enter!');
        const prevLine = this.doc.getLine(this.pos.line);

        const trimmedPrevLine = prevLine.trim();
        const indentSpaceMatches = prevLine.match(/^ +/);
        const indentSpace = indentSpaceMatches ? indentSpaceMatches[0] : '';
        const firstChar = trimmedPrevLine.charAt(0);

        let replacement = '';
        if (firstChar === '-' || firstChar === '*') {
          if (trimmedPrevLine === '-' || trimmedPrevLine === '*') {
            this.doc.getLineHandle(this.pos.line).text = '';
            replacement = `\n${indentSpace}`;
          } else {
            replacement = `\n${indentSpace}${firstChar} `;
          }
        } else {
          replacement = `\n${indentSpace}`;
        }

        this.cm.replaceSelection(replacement);
      },
    });

    this.cm.on('inputRead', (_cm: Editor, changeObj: any) => {
      // console.log('changeObj: ', changeObj);
      const inputChar = changeObj.text[0];
      // console.log('inputChar: ', inputChar);

      [
        ['(', ')'],
        ['[', ']'],
        ['{', '}'],
      ].forEach((pair) => {
        if (inputChar === pair[0]) {
          this.cm.replaceSelection(pair[1], 'start');
        }
      });

      this.showHints(this.getPrevChars());
    });

    this.cm.on('keydown', (cm: Editor, event: KeyboardEvent) => {
      // console.log('keydown! event.code: ', event.code);
      this.doc = cm.getDoc();
      this.pos = this.doc.getCursor();

      if (this.isHintsDisplayed && this.isKeyForHints(event.code)) {
        new EventDispatcher('editor-hints:keydown', { code: event.code }).dispatch();
        event.preventDefault();
      }
    });

    this.element.addEventListener('compositionstart', () => {
      // console.log('compositionstart!');
      this.isComposing = true;
    });

    this.element.addEventListener('compositionend', () => {
      // console.log('compositionend! this.pos: ', this.pos);
      this.isComposing = false;
      this.showHints(this.getPrevChars());
    });

    document.addEventListener('editor:select-hint', (event: any) => {
      const { selectedNoteId, selectedNoteTitle } = event.detail;
      // console.log('selectedNoteId: ', selectedNoteId);
      // console.log('currentLine: ', this.getCurrentLine());
      // console.log('cursor.ch: ', this.pos.ch);
      const prevChars = this.getCurrentLine().slice(0, this.pos.ch);
      // console.log('prevChars: ', prevChars);
      const startBracketMatch = this.reverseString(prevChars).match(/\[/);
      // console.log('startBracketMatch: ', startBracketMatch);
      let startBracketIndex: number = -1;
      if (typeof startBracketMatch?.index === 'number') {
        startBracketIndex = this.pos.ch - (startBracketMatch.index + 1);
        // console.log('startBracketIndex: ', startBracketIndex);
      }

      const afterChars = this.getCurrentLine().slice(this.pos.ch);
      // console.log('afterChars: ', afterChars);
      const endBracketMatch = afterChars.match(/]/);
      // console.log('endBracketMatch: ', endBracketMatch);
      let endBracketIndex: number = -1;
      if (typeof endBracketMatch?.index === 'number') {
        endBracketIndex = this.pos.ch + endBracketMatch.index;
        // console.log('endBracketIndex: ', endBracketIndex);
      }

      if (startBracketIndex >= 0 && endBracketIndex >= 0) {
        this.doc.replaceRange(
          `[${selectedNoteTitle}](/notes/${selectedNoteId})`,
          { line: this.pos.line, ch: startBracketIndex },
          { line: this.pos.line, ch: endBracketIndex + 1 },
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

  getCurrentLine() {
    return this.doc.getLine(this.pos.line);
  }

  getPrevChars() {
    return this.getCurrentLine().slice(0, this.pos.ch);
  }

  showHints(prevChars: string) {
    const linkTitle = this.symbolValue(prevChars, '[');
    // console.log('linkTitle: ', linkTitle);

    if (!this.isComposing && linkTitle) {
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

          const cursorCoords = this.cm.cursorCoords(false, 'window');
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
  }
}

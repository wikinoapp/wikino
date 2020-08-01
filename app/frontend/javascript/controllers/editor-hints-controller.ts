import axios from 'axios';
import { Controller } from 'stimulus';
import { EventDispatcher } from '../utils/event-dispatcher';

export default class extends Controller {
  static targets = ['hint', 'newNoteName'];

  element!: HTMLElement;
  selectedHintIndex!: number;
  hintTargets!: [HTMLElement];
  newNoteNameTarget!: HTMLElement;

  initialize() {
    // console.log('hints!');
    this.selectedHintIndex = -1;
    document.addEventListener('editor-hints:keydown', (event: any) => {
      const { code } = event.detail;

      if (code === 'ArrowDown') {
        this.selectedHintIndex += 1;
      } else if (code === 'ArrowUp') {
        this.selectedHintIndex -= 1;
      }

      if (this.selectedHintIndex < 0) {
        this.selectedHintIndex = this.hintTargets.length - 1;
      } else if (this.selectedHintIndex >= this.hintTargets.length) {
        this.selectedHintIndex = 0;
      }

      this.hintTargets.forEach((hintElm) => {
        hintElm.classList.remove('active');
      });
      const selectedHintElm = this.hintTargets[this.selectedHintIndex];
      selectedHintElm.classList.add('active');
      // console.log('receive event!: ', code);
      // console.log('this.hintTargets: ', this.hintTargets);
      // console.log('this.selectedHintIndex: ', this.selectedHintIndex);

      const selectedNoteId = selectedHintElm.dataset.noteId;
      const selectedNoteName = selectedHintElm.dataset.noteName;
      const createNewNote = selectedHintElm.dataset.createNewNote;
      if (code === 'Enter') {
        if (selectedNoteId && selectedNoteName) {
          new EventDispatcher('editor:select-hint', { selectedNoteId, selectedNoteName }).dispatch();
        } else if (createNewNote) {
          const newNoteTitle = this.newNoteNameTarget.innerText;

          axios
            .post('/api/internal/notes', {
              note_title: newNoteTitle,
            })
            .then((res: any) => {
              const noteId = res.data.database_id;
              const noteTitle = res.data.title;

              new EventDispatcher('editor:created-new-note', { noteId, noteTitle }).dispatch();
            })
            .catch((err) => {
              console.log('err: ', err);
            });
        }

        event.preventDefault();
      }
    });

    document.addEventListener('editor-hints:hide', (_event: any) => {
      this.element.classList.add('d-none');
    });

    document.addEventListener('editor-hints:show', (event: any) => {
      const { linkName, cursorCoords } = event.detail;
      // console.log('cursorCoords: ', cursorCoords);

      axios
        .get('/api/internal/notes', {
          params: {
            q: linkName,
          },
        })
        .then((res: any) => {
          const hintsHtml = res.data ? res.data : null;
          // console.log('hintsHtml: ', hintsHtml);
          this.element.innerHTML = hintsHtml;
          this.newNoteNameTarget.innerText = linkName;

          this.element.style.left = `${cursorCoords.left - 10}px`;
          this.element.style.top = `${cursorCoords.top + 25}px`;

          this.element.classList.remove('d-none');
        })
        .catch((err: any) => {
          console.log('err: ', err);
        });
    });
  }
}

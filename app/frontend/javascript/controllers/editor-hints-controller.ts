import { Controller } from 'stimulus';
import { EventDispatcher } from '../utils/event-dispatcher';

export default class extends Controller {
  static targets = ['hint', 'newNoteName'];

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
          new EventDispatcher('editor:create-new-note', { newNoteName: this.newNoteNameTarget.innerText }).dispatch();
        }
        event.preventDefault();
      }
    });

    document.addEventListener('editor-hints:hide', (_event: any) => {
      this.element.classList.add('d-none');
    });

    document.addEventListener('editor-hints:show', (event: any) => {
      const newNoteName = event.detail.linkName;

      this.element.classList.remove('d-none');
      this.newNoteNameTarget.innerText = newNoteName;
    });
  }
}

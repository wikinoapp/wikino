import { Controller } from 'stimulus';

export default class extends Controller {
  element!: HTMLElement;

  initialize() {
    document.addEventListener('note-preview:update-preview', (event: any) => {
      const { bodyHtml } = event.detail;

      this.element.innerHTML = bodyHtml;
    });
  }
}

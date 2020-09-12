import { Controller } from 'stimulus';

export default class extends Controller {
  initialize() {
    document.addEventListener('note-preview:update', (event: any) => {
      const { bodyHtml } = event.detail;

      this.element.innerHTML = bodyHtml;
    });
  }
}

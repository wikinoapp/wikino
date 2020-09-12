import { Controller } from 'stimulus';

export default class extends Controller {
  initialize() {
    document.addEventListener('note-backlinks:update', (event: any) => {
      const { backlinksHtml } = event.detail;

      this.element.innerHTML = backlinksHtml;
    });
  }
}

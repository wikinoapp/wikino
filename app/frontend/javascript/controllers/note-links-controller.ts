import { Controller } from 'stimulus';

export default class extends Controller {
  initialize() {
    document.addEventListener('note-links:update', (event: any) => {
      const { linksHtml } = event.detail;

      this.element.innerHTML = linksHtml;
    });
  }
}

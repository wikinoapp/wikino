import { Controller } from 'stimulus';

export default class extends Controller {
  element!: HTMLElement;
  data!: any;

  initialize() {
    document.addEventListener('note-alert:update', (event: any) => {
      const { alertHtml } = event.detail;

      this.element.innerHTML = alertHtml;
    });
  }
}

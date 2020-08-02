import { Controller } from 'stimulus';

export default class extends Controller {
  element!: HTMLElement;
  data!: any;

  initialize() {
    document.addEventListener('note-time:update-time', (event: any) => {
      const { updatedAt } = event.detail;

      this.element.innerText = updatedAt;
    });
  }

  connect() {
    this.element.innerText = this.data.get('initTime');
  }
}

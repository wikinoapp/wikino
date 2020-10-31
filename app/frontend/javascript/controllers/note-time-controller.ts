import { Controller } from 'stimulus';

export default class extends Controller {
  element!: HTMLElement;
  data!: any;

  initialize() {
    document.addEventListener('note-time:update', (event: any) => {
      const { modifiedAt } = event.detail;

      this.element.innerText = modifiedAt;
    });
  }

  connect() {
    this.element.innerText = this.data.get('initTime');
  }
}

import { Controller } from "@hotwired/stimulus";

export default class extends Controller {
  static targets = ["dialog", "trigger"];

  declare readonly dialogTarget: HTMLDialogElement;
  declare readonly triggerTarget: HTMLElement;

  open() {
    this.dialogTarget.showModal();
  }

  close(event?: Event) {
    if (event) {
      event.preventDefault();
    }

    this.dialogTarget.close();
  }
}

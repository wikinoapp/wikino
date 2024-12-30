import { Controller } from "@hotwired/stimulus";

export default class extends Controller<HTMLFormElement> {
  static targets = ["actionButton"];

  declare readonly actionButtonTargets: HTMLButtonElement[];

  disableAllActionButtons(event) {
    this.actionButtonTargets.filter((target) => event.target !== target).forEach((button) => {
      button.disabled = true;
    });
  }
}

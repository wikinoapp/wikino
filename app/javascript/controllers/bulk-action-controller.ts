import { Controller, ActionEvent } from "@hotwired/stimulus";

export default class extends Controller<HTMLFormElement> {
  static targets = ["draftSaveButton"];

  declare readonly draftSaveButtonTarget: HTMLInputElement;

  toggleCheck(event: ActionEvent) {
    console.log("!!! bulk-action#toggleCheck > event: ", event);
    this.dispatch("check", { detail: { payload: event.params.payload } })
  }
}

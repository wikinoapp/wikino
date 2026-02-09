import { Controller } from "@hotwired/stimulus";
import { useDebounce } from "stimulus-use";

export default class extends Controller<HTMLFormElement> {
  static debounces = [
    {
      name: "saveAsDraft",
      wait: 500,
    },
  ];
  static targets = ["draftSaveButton"];

  declare readonly draftSaveButtonTarget: HTMLInputElement;

  connect() {
    useDebounce(this);
  }

  saveAsDraft() {
    this.element.requestSubmit(this.draftSaveButtonTarget);
  }
}

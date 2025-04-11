import { Controller } from "@hotwired/stimulus";

export default class extends Controller<HTMLElement> {
  static values = {
    status: String,
  }

  declare readonly statusValue: string;

  connect() {
    if (this.statusValue === "succeeded") {
      window.location.reload();
    }
  }
}

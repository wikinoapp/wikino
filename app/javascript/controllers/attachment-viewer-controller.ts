import { Controller } from "@hotwired/stimulus";

export default class AttachmentViewerController extends Controller<HTMLElement> {
  static values = {
    url: String
  };

  declare readonly urlValue: string;

  open(event: Event) {
    event.preventDefault();
    
    // 新規タブで画像を開く
    window.open(this.urlValue, "_blank", "noopener,noreferrer");
  }
}
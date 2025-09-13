import { Controller } from "@hotwired/stimulus";

export default class extends Controller {
  static targets = ["panel", "overlay"];

  declare readonly panelTarget: HTMLElement;
  declare readonly overlayTarget: HTMLElement;

  open(): void {
    // サイドバーを表示
    this.panelTarget.classList.remove("-translate-x-full");
    this.panelTarget.classList.add("translate-x-0");

    // オーバーレイを表示
    this.overlayTarget.classList.remove("hidden");

    // スクロールを無効化
    document.body.style.overflow = "hidden";
  }

  close(): void {
    // サイドバーを非表示
    this.panelTarget.classList.add("-translate-x-full");
    this.panelTarget.classList.remove("translate-x-0");

    // オーバーレイを非表示
    this.overlayTarget.classList.add("hidden");

    // スクロールを有効化
    document.body.style.overflow = "";
  }

  toggle(): void {
    if (this.panelTarget.classList.contains("-translate-x-full")) {
      this.open();
    } else {
      this.close();
    }
  }
}

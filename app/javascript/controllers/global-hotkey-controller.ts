import { Controller } from "@hotwired/stimulus";
import { install } from "@github/hotkey";

// グローバルホットキーを処理するコントローラー
export default class extends Controller {
  static targets = ["searchPath"];
  declare readonly searchPathTarget: HTMLMetaElement;

  connect() {
    // `s` キーまたは `/` キーで検索ページにアクセス
    install(this.element, "s,/");
    this.element.addEventListener("hotkey-fire", this.navigateToSearch);
  }

  disconnect() {
    this.element.removeEventListener("hotkey-fire", this.navigateToSearch);
  }

  private navigateToSearch = (event: Event) => {
    event.preventDefault();

    // 検索ページのパスを取得
    const searchPath = this.searchPathTarget.content;

    // 検索ページに移動
    window.location.href = searchPath;
  };
}

import { Controller } from "@hotwired/stimulus";
import { install } from "@github/hotkey";

// グローバルホットキーを処理するコントローラー
export default class extends Controller {
  static targets = ["searchPath"];
  static values = { currentSpaceIdentifier: String };

  declare readonly searchPathTarget: HTMLMetaElement;
  declare currentSpaceIdentifierValue: string;

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
    let searchPath = this.searchPathTarget.content;

    // 現在のスペース内にいる場合はspace:フィルターを付与
    if (this.currentSpaceIdentifierValue) {
      const url = new URL(searchPath, window.location.origin);
      url.searchParams.set("q", `space:${this.currentSpaceIdentifierValue}`);
      searchPath = url.pathname + url.search;
    }

    // 検索ページに移動
    window.location.href = searchPath;
  };
}

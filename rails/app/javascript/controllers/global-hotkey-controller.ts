import { Controller } from "@hotwired/stimulus";
import { install } from "@github/hotkey";

// グローバルホットキーを処理するコントローラー
export default class extends Controller {
  static targets = ["searchPath"];

  declare readonly searchPathTarget: HTMLMetaElement;

  connect() {
    // `s` キーまたは `/` キーで検索ページにアクセス
    install(this.element as HTMLElement, "s,/");
    this.element.addEventListener("hotkey-fire", this.navigateToSearch);
  }

  disconnect() {
    this.element.removeEventListener("hotkey-fire", this.navigateToSearch);
  }

  private navigateToSearch = (event: Event) => {
    event.preventDefault();

    // 入力フィールドにフォーカスがある場合は何もしない
    const activeElement = document.activeElement;
    if (activeElement && this.isInputElement(activeElement)) {
      return;
    }

    // 検索ページのパスを取得
    const searchPath = this.searchPathTarget.content;

    // 検索ページに移動
    window.location.href = searchPath;
  };

  // 入力可能な要素かどうかを判定
  private isInputElement(element: Element): boolean {
    const tagName = element.tagName.toLowerCase();

    // input, textarea, select要素の場合
    if (tagName === "input" || tagName === "textarea" || tagName === "select") {
      return true;
    }

    // contenteditable属性がtrueの要素の場合
    if (element.getAttribute("contenteditable") === "true") {
      return true;
    }

    // CodeMirrorエディタの場合（.cm-contentクラスを持つ要素）
    if (element.classList.contains("cm-content")) {
      return true;
    }

    return false;
  }
}

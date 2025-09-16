import { Controller } from "@hotwired/stimulus";

export default class extends Controller<HTMLElement> {
  declare observer: IntersectionObserver;

  connect() {
    // スティッキーヘッダーの表示状態を管理するためのオブザーバーを設定
    // 要素が画面上端に固定されたときにdata-stuck属性を付与し、CSSで表示を切り替える
    this.observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          // intersectionRatio < 1: 要素の一部が画面外に出ている（スティッキー状態）
          // intersectionRatio = 1: 要素が完全に画面内に表示されている（通常状態）
          if (entry.intersectionRatio < 1) {
            this.element.dataset.stuck = "";
          } else {
            delete this.element.dataset.stuck;
          }
        });
      },
      {
        // rootMargin: -1px により、要素が画面上端から1px上に移動したタイミングを検出
        rootMargin: "-1px 0px 0px 0px",
        // threshold: 1 で要素が完全に表示されているかどうかを判定
        threshold: 1,
      },
    );

    this.observer.observe(this.element);
  }

  disconnect() {
    // Stimulusコントローラーがアンマウントされる際のクリーンアップ処理
    // IntersectionObserverを停止してメモリリークを防ぐ
    if (this.observer) {
      this.observer.disconnect();
    }
  }
}

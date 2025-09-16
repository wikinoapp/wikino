import { Controller } from "@hotwired/stimulus";

export default class extends Controller<HTMLElement> {
  declare observer: IntersectionObserver;

  connect() {
    // IntersectionObserverのみを使用してスティッキー状態を検出
    // スクロールイベントハンドラを削除してパフォーマンスを改善
    this.observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          // 要素が完全に表示されているかどうかをチェック
          // threshold: 1のみを使用してパフォーマンスを最適化
          if (entry.intersectionRatio < 1) {
            this.element.dataset.stuck = "";
          } else {
            delete this.element.dataset.stuck;
          }
        });
      },
      {
        // 要素が画面上端に到達したときを検出
        rootMargin: "-1px 0px 0px 0px",
        // 単一のthresholdでパフォーマンスを改善
        threshold: 1,
      },
    );

    this.observer.observe(this.element);
  }

  disconnect() {
    // クリーンアップ：要素が削除されるときにobserverを停止
    if (this.observer) {
      this.observer.disconnect();
    }
  }
}

import { Controller } from "@hotwired/stimulus";

export default class extends Controller<HTMLElement> {
  declare observer: IntersectionObserver;

  connect() {
    // IntersectionObserverを使用してスティッキー状態を検出
    // rootMargin: "-1px 0px 0px 0px"により、要素が画面上端を1px超えたときに検出
    this.observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          // intersectionRatio < 1 は要素が部分的に隠れている（スティッキー状態）
          // boundingClientRect.top <= 0 は要素が画面上端以上にある
          if (!entry.isIntersecting || (entry.intersectionRatio < 1 && entry.boundingClientRect.top <= 0)) {
            this.element.dataset.stuck = "";
          } else {
            delete this.element.dataset.stuck;
          }
        });
      },
      {
        // 要素が画面上端に到達したときを正確に検出
        rootMargin: "-1px 0px 0px 0px",
        // より細かいthresholdで中間状態も検出
        threshold: [0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1],
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

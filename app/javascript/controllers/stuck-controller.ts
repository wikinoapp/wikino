import { Controller } from "@hotwired/stimulus";

export default class extends Controller<HTMLElement> {
  declare observer: IntersectionObserver;
  declare scrollHandler: () => void;
  declare scrollTimeout: number;

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

    // スクロールイベントでフォールバック処理を追加
    this.scrollHandler = () => {
      // throttleのためのタイムアウト処理
      if (this.scrollTimeout) {
        return;
      }

      this.scrollTimeout = window.setTimeout(() => {
        this.scrollTimeout = 0;
        this.checkStuckState();
      }, 10); // 10msごとにチェック
    };

    window.addEventListener("scroll", this.scrollHandler, { passive: true });

    // 初期状態をチェック
    this.checkStuckState();
  }

  private checkStuckState() {
    const rect = this.element.getBoundingClientRect();
    const computedStyle = window.getComputedStyle(this.element);

    // position: stickyの要素が実際にスティッキー状態かを判定
    // 要素の上端が画面上端付近（1px以内）にあり、position: stickyの場合
    if (computedStyle.position === "sticky" && rect.top <= 1) {
      this.element.dataset.stuck = "";
    } else if (rect.top > 1) {
      delete this.element.dataset.stuck;
    }
  }

  disconnect() {
    // クリーンアップ：要素が削除されるときにobserverを停止
    if (this.observer) {
      this.observer.disconnect();
    }

    // スクロールイベントのクリーンアップ
    if (this.scrollHandler) {
      window.removeEventListener("scroll", this.scrollHandler);
    }

    // タイムアウトのクリーンアップ
    if (this.scrollTimeout) {
      window.clearTimeout(this.scrollTimeout);
    }
  }
}

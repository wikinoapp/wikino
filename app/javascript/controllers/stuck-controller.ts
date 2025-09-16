import { Controller } from "@hotwired/stimulus";

export default class extends Controller<HTMLElement> {
  declare isStuck: boolean;
  declare scrollHandler: () => void;
  declare rafId: number;

  connect() {
    this.isStuck = false;

    // スクロールイベントのみを使用（IntersectionObserverは削除）
    this.scrollHandler = () => {
      // requestAnimationFrameを使用してスムーズな更新
      if (this.rafId) {
        cancelAnimationFrame(this.rafId);
      }
      
      this.rafId = requestAnimationFrame(() => {
        this.checkStuckState();
      });
    };

    window.addEventListener("scroll", this.scrollHandler, { passive: true });

    // 初期状態をチェック
    this.checkStuckState();
  }

  private checkStuckState() {
    const rect = this.element.getBoundingClientRect();
    const shouldBeStuck = rect.top <= 0;

    // 状態が変わったときだけDOM操作を行う
    if (shouldBeStuck && !this.isStuck) {
      this.element.dataset.stuck = "";
      this.isStuck = true;
    } else if (!shouldBeStuck && this.isStuck) {
      delete this.element.dataset.stuck;
      this.isStuck = false;
    }
  }

  disconnect() {
    // スクロールイベントのクリーンアップ
    if (this.scrollHandler) {
      window.removeEventListener("scroll", this.scrollHandler);
    }

    // rafのクリーンアップ
    if (this.rafId) {
      cancelAnimationFrame(this.rafId);
    }
  }
}

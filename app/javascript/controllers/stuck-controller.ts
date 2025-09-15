import { Controller } from "@hotwired/stimulus";

export default class extends Controller<HTMLElement> {
  declare observer: IntersectionObserver;
  declare sentinel: HTMLElement;

  connect() {
    // sticky要素の直前にセンチネル要素を挿入
    this.sentinel = document.createElement("div");
    this.sentinel.style.position = "absolute";
    this.sentinel.style.top = "-1px";
    this.sentinel.style.left = "0";
    this.sentinel.style.right = "0";
    this.sentinel.style.height = "1px";
    this.sentinel.style.pointerEvents = "none";
    this.element.parentElement?.insertBefore(this.sentinel, this.element);

    // Intersection Observerを設定
    this.observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          // センチネルが見えなくなった = sticky要素がstickされた
          if (!entry.isIntersecting) {
            this.element.dataset.stuck = "";
          } else {
            delete this.element.dataset.stuck;
          }
        });
      },
      {
        threshold: 0,
        rootMargin: "0px"
      }
    );

    this.observer.observe(this.sentinel);
  }

  disconnect() {
    if (this.observer) {
      this.observer.disconnect();
    }
    if (this.sentinel && this.sentinel.parentElement) {
      this.sentinel.remove();
    }
  }
}
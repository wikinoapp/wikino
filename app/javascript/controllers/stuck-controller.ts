import { Controller } from "@hotwired/stimulus";

export default class extends Controller<HTMLElement> {
  declare ticking: boolean;

  connect() {
    this.ticking = false;

    window.addEventListener("scroll", this.handleScroll.bind(this), { passive: true });
  }

  handleScroll() {
    if (!this.ticking) {
      window.requestAnimationFrame(() => {
        const elementTop = this.element.getBoundingClientRect().top;
        const currentScrollPos = window.scrollY;

        if (elementTop <= 0 && currentScrollPos > 0) {
          this.element.dataset.stuck = "";
        } else {
          delete this.element.dataset.stuck;
        }

        this.ticking = false;
      });

      this.ticking = true;
    }
  }
}

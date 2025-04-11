import { Controller } from "@hotwired/stimulus";

export default class extends Controller {
  static values = {
    status: String,
  }

  declare readonly statusValue: string;
  private intervalId: number | null = null;

  connect() {
    if (this.statusValue !== "succeeded" && this.statusValue !== "failed") {
      // 1秒ごとにリロードする
      this.intervalId = setInterval(() => {
        this.element.reload();
      }, 1000);
    }
  }

  disconnect() {
    // コントローラーが切断されたらインターバルをクリアする
    if (this.intervalId !== null) {
      clearInterval(this.intervalId);
      this.intervalId = null;
    }
  }
}

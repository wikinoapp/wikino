import { Controller } from '@hotwired/stimulus';

export default class extends Controller<HTMLElement> {
  static values = {
    utcTime: String,
  };

  declare timezone: string;
  declare utcTimeValue: string;

  initialize() {
    this.timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  }

  connect() {
    const localTime = this.localTime();

    // https://daisyui.com/components/tooltip/
    this.element.classList.add('tooltip');
    this.element.dataset.tip = localTime;
  }

  localTime() {
    return new Date(this.utcTimeValue).toLocaleString('ja-JP', {
      timeZone: this.timezone,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
  }
}

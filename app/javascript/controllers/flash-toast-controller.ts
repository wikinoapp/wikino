import { Controller } from '@hotwired/stimulus';

const animationClasses = ['scale-90', 'opacity-0'];
const hiddenClass = 'hidden';

export default class extends Controller {
  static targets = ['alert', 'noticeIcon', 'alertIcon', 'message'];
  static values = {
    type: String,
    messageHtml: String,
  };

  declare readonly messageHtmlValue: string;
  declare readonly typeValue: string;

  declare readonly noticeIconTarget: HTMLSpanElement;
  declare readonly alertIconTarget: HTMLSpanElement;
  declare readonly messageTarget: HTMLSpanElement;

  messageHtml!: string;
  type!: string;

  connect() {
    this.type = this.typeValue || 'notice';
    this.messageHtml = this.messageHtmlValue || '';

    if (this.type && this.messageHtml) {
      this.showFlashToast();
    }

    document.addEventListener('flash-toast:show', ({ detail: { type, messageHtml } }: any) => {
      this.type = type || 'notice';
      this.messageHtml = messageHtml || '';

      this.showFlashToast();
    });
  }

  showFlashToast() {
    this.showFlashIcon()
    this.messageTarget.innerHTML = this.messageHtml;
    this.element.classList.remove(hiddenClass, ...animationClasses);

    if (this.type === 'notice') {
      setTimeout(() => {
        this.hideFlashToast();
      }, 2000);
    }
  }

  showFlashIcon() {
    if (this.type === 'alert') {
      this.noticeIconTarget.classList.add(hiddenClass);
      this.alertIconTarget.classList.remove(hiddenClass);
    } else {
      this.noticeIconTarget.classList.remove(hiddenClass);
      this.alertIconTarget.classList.add(hiddenClass);

    }
  }

  hideFlashToast() {
    this.element.classList.add(...animationClasses);

    setTimeout(() => {
      this.element.classList.add(hiddenClass);
    }, 150);
  }
}

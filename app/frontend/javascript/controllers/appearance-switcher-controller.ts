import Cookies from 'js-cookie';
import { Controller } from 'stimulus';

export default class extends Controller {
  element!: HTMLElement;

  setLight(event: any) {
    this.resetActive();
    event.target.classList.add('active');
    this.setColorScheme('light');
  }

  setDark(event: any) {
    this.resetActive();
    event.target.classList.add('active');
    this.setColorScheme('dark');
  }

  setSystemDefault(event: any) {
    this.resetActive();
    event.target.classList.add('active');
    this.setColorScheme('system-default');
  }

  resetActive() {
    const elms: any = this.element.getElementsByClassName('active');

    for (let elm of elms) {
      elm.classList.remove('active');
    }
  }

  setColorScheme(name: string) {
    this.data.set('colorScheme', name);
    Cookies.set('color_scheme', name);
  }
}

import axios from 'axios';
import ujs from '@rails/ujs';
import { Application } from 'stimulus';
import { definitionsFromContext } from 'stimulus/webpack-helpers';
import Turbolinks from 'turbolinks';

document.addEventListener('turbolinks:load', (_event) => {
  axios.defaults.headers.common['X-CSRF-Token'] = document
    .querySelector('meta[name="csrf-token"]')
    ?.getAttribute('content');
});

const application = Application.start();
const context = (require as any).context('./controllers', true, /\.ts$/);
application.load(definitionsFromContext(context));

ujs.start();
Turbolinks.start();

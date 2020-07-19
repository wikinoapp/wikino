import ujs from '@rails/ujs';
import { Application } from 'stimulus';
import { definitionsFromContext } from 'stimulus/webpack-helpers';
import Turbolinks from 'turbolinks';

document.addEventListener('turbolinks:load', (_event) => {
});

const application = Application.start();
const context = (require as any).context('./controllers', true, /\.ts$/);
application.load(definitionsFromContext(context));

ujs.start();
Turbolinks.start();

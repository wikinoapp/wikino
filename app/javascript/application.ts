import '@hotwired/turbo';

import { Application } from '@hotwired/stimulus';

import FlashToastController from './controllers/flash-toast-controller';

const application = Application.start();
application.debug = false;
window.Stimulus = application;

Stimulus.register('flash-toast', FlashToastController);

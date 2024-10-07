import '@hotwired/turbo';

import { Application } from '@hotwired/stimulus';

import FlashToastController from './controllers/flash-toast-controller';
import PageEditorController from './controllers/page-editor-controller';
import PageEditorFormController from './controllers/page-editor-form-controller';

declare global {
  interface Window {
    Stimulus: Application;
  }
}

const application = Application.start();
application.debug = false;
window.Stimulus = application;

window.Stimulus.register('flash-toast', FlashToastController);
window.Stimulus.register('page-editor-form', PageEditorFormController);
window.Stimulus.register('page-editor', PageEditorController);

import '@hotwired/turbo';

import { Application } from '@hotwired/stimulus';

import FlashToastController from './controllers/flash-toast-controller';
import NoteEditorFormController from './controllers/note-editor-form-controller';

declare global {
  interface Window {
    Stimulus: Application;
  }
}

const application = Application.start();
application.debug = false;
window.Stimulus = application;

window.Stimulus.register('flash-toast', FlashToastController);
window.Stimulus.register('note-editor-form', NoteEditorFormController);

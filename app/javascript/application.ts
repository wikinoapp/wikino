import 'bootstrap';
import { Application } from '@hotwired/stimulus';
import * as Turbo from '@hotwired/turbo';
import ujs from '@rails/ujs';

ujs.start();
window.Stimulus = Application.start();

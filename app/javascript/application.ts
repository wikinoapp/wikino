import { Application } from '@hotwired/stimulus';
import * as Turbo from '@hotwired/turbo';
import ujs from '@rails/ujs';
import 'bootstrap';
import 'trix';

ujs.start();
window.Stimulus = Application.start();

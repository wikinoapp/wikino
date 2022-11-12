import debounce from 'lodash/debounce';
import Trix from 'trix';

import fetcher from './utils/fetcher';

document.addEventListener('trix-before-initialize', () => {
  // ツールバーを無効化する
  Trix.config.toolbar.getDefaultHTML = () => null;
});

let isSaving = false;

document.addEventListener(
  'trix-change',
  debounce(async (event: any) => {
    const { target } = event;
    const { noteId } = target.dataset;
    const body = target.editor.getDocument().toString();

    if (!isSaving) {
      isSaving = true;

      try {
        await fetcher.patch(`/api/internal/notes/${noteId}`, {
          note: {
            body,
          },
        });
      } catch (err) {
        console.error(err);
      } finally {
        isSaving = false;
      }
    }
  }, 1000),
);

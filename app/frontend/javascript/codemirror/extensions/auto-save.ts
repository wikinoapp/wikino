import axios from 'axios';
import { Extension } from '@codemirror/next/state';
import { EditorView } from '@codemirror/next/view';
import debounce from 'lodash/debounce';

import { EventDispatcher } from '../../utils/event-dispatcher';

export function autoSave(noteDatabaseId: string): Extension {
  let isSaving = false;

  return EditorView.domEventHandlers({
    keyup: debounce((event: any, view: EditorView) => {
      const noteBody = view.state.doc.toString();

      if (!isSaving) {
        isSaving = true;

        axios
          .patch(`/api/internal/notes/${noteDatabaseId}`, {
            note: {
              body: noteBody,
            },
          })
          .then((res: any) => {
            const { note, linksHtml, backlinksHtml } = res.data;
            const { bodyHtml, updatedAt } = note;

            new EventDispatcher('note-time:update', { updatedAt }).dispatch();
            new EventDispatcher('note-preview:update', { bodyHtml }).dispatch();
            new EventDispatcher('note-links:update', { linksHtml }).dispatch();
            new EventDispatcher('note-backlinks:update', { backlinksHtml }).dispatch();
          })
          .catch((err) => {
            console.log('err: ', err);
          })
          .then(() => {
            isSaving = false;
          });
      }
    }, 300),
  });
}

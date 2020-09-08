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
            new EventDispatcher('note-time:update-time', { updatedAt: res.data.updated_at }).dispatch();
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

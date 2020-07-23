# frozen_string_literal: true

module NoteDetail
  class FetchNoteRepository < ApplicationRepository
    def fetch(note_id:)
      result = execute(variables: { noteId: note_id })
      note_node = result.to_h.dig("data", "node")

      raise ActiveRecord::RecordNotFound unless note_node

      NoteEntity.from_node(note_node)
    end
  end
end

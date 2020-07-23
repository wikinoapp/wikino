# frozen_string_literal: true

module NoteDetail
  class FetchNoteRepository < ApplicationRepository
    def fetch(note_number:)
      result = execute(variables: { noteNumber: note_number })
      note_node = result.to_h.dig("data", "viewer", "team", "note")

      raise ActiveRecord::RecordNotFound unless note_node

      NoteEntity.from_node(note_node)
    end
  end
end

# frozen_string_literal: true

module EditNote
  class FetchNoteRepository < ApplicationRepository
    def call(database_id:)
      result = execute(variables: { databaseId: database_id })
      note_node = result.to_h.dig("data", "viewer", "note")

      raise ActiveRecord::RecordNotFound unless note_node

      NoteEntity.from_node(note_node)
    end
  end
end

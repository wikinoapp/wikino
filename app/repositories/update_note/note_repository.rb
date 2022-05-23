# typed: false
# frozen_string_literal: true

module UpdateNote
  class NoteRepository < ApplicationRepository
    def call(database_id:)
      result = query(variables: { databaseId: database_id })
      note_node = result.to_h.dig("data", "viewer", "note")

      raise ActiveRecord::RecordNotFound unless note_node

      NoteEntity.from_node(note_node)
    end
  end
end

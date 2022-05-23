# typed: false
# frozen_string_literal: true

class UpdateNoteRepository < ApplicationRepository
  def call(id:, body:)
    result = mutate(variables: {
      id: id,
      body: body
    })

    note_node = result.dig("data", "updateNote", "note")
    error_node = result.dig("data", "updateNote", "error")

    [NoteEntity.from_node(note_node), error_node&.dig("code")]
  end
end

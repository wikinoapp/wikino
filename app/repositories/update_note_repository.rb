# frozen_string_literal: true

class UpdateNoteRepository < ApplicationRepository
  def call(id:, body:)
    result = execute(variables: {
      id: id,
      body: body
    })

    error_nodes = result.dig("data", "updateNote", "errors")
    if error_nodes.present?
      return [nil, MutationErrorEntity.from_nodes(error_nodes)]
    end

    note_node = result.dig("data", "updateNote", "note")

    [NoteEntity.from_node(note_node), nil]
  end
end

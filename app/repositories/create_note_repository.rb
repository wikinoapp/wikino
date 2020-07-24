# frozen_string_literal: true

class CreateNoteRepository < ApplicationRepository
  def call(params:)
    result = execute(variables: {
      body: params[:body]
    })

    error_nodes = result.dig("data", "createNote", "errors")
    if error_nodes.present?
      return [nil, MutationErrorEntity.from_nodes(error_nodes)]
    end

    note_node = result.dig("data", "createNote", "note")

    [NoteEntity.from_node(note_node), nil]
  end
end

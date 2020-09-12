# frozen_string_literal: true

class CreateNoteRepository < ApplicationRepository
  def call(params: {})
    variables = {}

    if body = params[:body]
      variables[:body] = body
    end

    result = mutate(variables: variables)

    error_nodes = result.dig("data", "createNote", "errors")
    if error_nodes.present?
      return [nil, MutationErrorEntity.from_nodes(error_nodes)]
    end

    note_node = result.dig("data", "createNote", "note")

    [NoteEntity.from_node(note_node), nil]
  end
end

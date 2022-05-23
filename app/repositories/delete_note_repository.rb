# typed: false
# frozen_string_literal: true

class DeleteNoteRepository < ApplicationRepository
  def call(database_id:)
    result = mutate(variables: {
      id: NonotoSchema.id_from_object(Note.find(database_id), Note)
    })

    error_nodes = result.dig("data", "deleteNote", "errors")
    if error_nodes.present?
      return [nil, MutationErrorEntity.from_nodes(error_nodes)]
    end

    [nil, nil]
  end
end

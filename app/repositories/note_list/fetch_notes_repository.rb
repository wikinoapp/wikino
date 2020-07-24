# frozen_string_literal: true

module NoteList
  class FetchNotesRepository < ApplicationRepository
    def call(pagination:)
      result = execute(variables: {
        first: pagination.first,
        last: pagination.last,
        before: pagination.before,
        after: pagination.after
      })
      data = result.to_h.dig("data", "viewer", "notes")

      [NoteEntity.from_nodes(data["nodes"]), PageInfoEntity.from_node(data["pageInfo"])]
    end
  end
end

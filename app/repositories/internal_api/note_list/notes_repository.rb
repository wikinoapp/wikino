# typed: false
# frozen_string_literal: true

module InternalApi
  module NoteList
    class NotesRepository < ApplicationRepository
      def call(q:)
        result = query(variables: { q: q || "" })
        data = result.to_h.dig("data", "viewer", "notes")

        NoteEntity.from_nodes(data["nodes"])
      end
    end
  end
end

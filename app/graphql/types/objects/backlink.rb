# typed: true
# frozen_string_literal: true

module Types
  module Objects
    class Backlink < Types::Objects::Base
      implements GraphQL::Types::Relay::Node

      field :note, Types::Objects::Note, null: false
      field :title, String, null: false

      def title
        note.then(&:title)
      end

      def note
        RecordLoader.for(Note).load(object.note_id)
      end
    end
  end
end

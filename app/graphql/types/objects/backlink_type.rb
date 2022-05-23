# typed: true
# frozen_string_literal: true

module Types
  module Objects
    class BacklinkType < Types::Objects::Base
      implements GraphQL::Types::Relay::Node

      field :title, String, null: false
      field :note, Types::Objects::NoteType, null: false

      def title
        note.then(&:title)
      end

      def note
        RecordLoader.for(Note).load(object.note_id)
      end
    end
  end
end

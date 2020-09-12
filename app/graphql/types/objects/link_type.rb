# frozen_string_literal: true

module Types
  module Objects
    class LinkType < Types::Objects::Base
      implements GraphQL::Types::Relay::Node

      field :note, Types::Objects::NoteType, null: false

      def note
        RecordLoader.for(Note).load(object.target_note_id)
      end
    end
  end
end

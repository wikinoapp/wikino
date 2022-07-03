# typed: strict
# frozen_string_literal: true

module Types
  module Objects
    class Backlink < Types::Objects::Base
      extend T::Sig

      description "A backlink."

      implements GraphQL::Types::Relay::Node

      field :note, Types::Objects::Note, "A note.", null: false
      field :title, String, "A title of note.", null: false

      sig { returns(Promise) }
      def title
        note.then(&:title)
      end

      sig { returns(Promise) }
      def note
        RecordLoader.for(::Note).load(object.note_id)
      end
    end
  end
end

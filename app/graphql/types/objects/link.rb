# typed: strict
# frozen_string_literal: true

module Types
  module Objects
    class Link < Types::Objects::Base
      extend T::Sig

      implements GraphQL::Types::Relay::Node

      field :note, Types::Objects::Note, null: false

      sig { returns(Promise) }
      def note
        RecordLoader.for(::Note).load(object.target_note_id)
      end
    end
  end
end

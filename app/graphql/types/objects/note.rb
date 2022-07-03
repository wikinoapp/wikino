# typed: strict
# frozen_string_literal: true

module Types
  module Objects
    class Note < Types::Objects::Base
      extend T::Sig

      description "A note."

      implements GraphQL::Types::Relay::Node

      global_id_field :id

      field :backlinks, Types::Objects::Backlink.connection_type, "A list of backlinks associated with the note.", null: false
      field :content, Types::Objects::NoteContent, "A content of the note.", null: false
      field :database_id, String, "Identifies the primary key from the database.", null: false
      field :links, Types::Objects::Link.connection_type, "A list of links associated with the note.", null: false
      field :modified_at, GraphQL::Types::ISO8601DateTime, "Identifies the date and time when the object was modified.", null: true
      field :title, String, "A title of the note.", null: false

      sig { returns(Promise) }
      def backlinks
        AssociationLoader.for(::Note, :backlinks).load(object)
      end

      sig { returns(Promise) }
      def content
        RecordLoader.for(::Note).load(object.id)
      end
    end
  end
end

# typed: true
# frozen_string_literal: true

module Types
  module Objects
    class Note < Types::Objects::Base
      implements GraphQL::Types::Relay::Node

      global_id_field :id

      field :database_id, String, null: false
      field :title, String, null: false
      field :content, Types::Objects::NoteContent, null: false
      field :modified_at, GraphQL::Types::ISO8601DateTime, null: true
      field :links, Types::Objects::Link.connection_type, null: false
      field :backlinks, Types::Objects::Backlink.connection_type, null: false

      def backlinks
        AssociationLoader.for(Note, :backlinks).load(object)
      end
    end
  end
end

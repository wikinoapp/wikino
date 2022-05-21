# frozen_string_literal: true

module Types
  module Objects
    class NoteType < Types::Objects::Base
      implements GraphQL::Types::Relay::Node

      global_id_field :id

      field :database_id, String, null: false
      field :title, String, null: false
      field :content, String, null: false
      field :content_html, String, null: false
      field :modified_at, GraphQL::Types::ISO8601DateTime, null: true
      field :links, Types::Objects::LinkType.connection_type, null: false
      field :backlinks, Types::Objects::BacklinkType.connection_type, null: false

      def backlinks
        AssociationLoader.for(Note, :backlinks).load(object)
      end
    end
  end
end

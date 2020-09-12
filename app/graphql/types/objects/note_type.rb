# frozen_string_literal: true

module Types
  module Objects
    class NoteType < Types::Objects::Base
      implements GraphQL::Types::Relay::Node

      global_id_field :id

      field :database_id, String, null: false
      field :title, String, null: false
      field :body, String, null: false
      field :body_html, String, null: false
      field :cover_image_url, String, null: true
      field :updated_at, GraphQL::Types::ISO8601DateTime, null: false
      field :links, Types::Objects::LinkType.connection_type, null: false
      field :backlinks, Types::Objects::BacklinkType.connection_type, null: false
    end
  end
end

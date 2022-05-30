# typed: strict
# frozen_string_literal: true

module Types
  module Objects
    class NoteContent < Types::Objects::Base
      implements GraphQL::Types::Relay::Node

      global_id_field :id

      field :body, String, null: false
      field :body_html, String, null: false
    end
  end
end

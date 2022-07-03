# typed: strict
# frozen_string_literal: true

module Types
  module Objects
    class NoteContent < Types::Objects::Base
      description "A content of the note."

      implements GraphQL::Types::Relay::Node

      global_id_field :id

      field :body, String, "A body of the content.", null: false
      field :body_html, String, "A body of the content which is rendered as HTML.", null: false
    end
  end
end

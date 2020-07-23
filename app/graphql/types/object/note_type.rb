# frozen_string_literal: true

module Types
  module Object
    class NoteType < Types::Object::Base
      implements GraphQL::Types::Relay::Node

      global_id_field :id

      field :title, String, null: false
      field :body, String, null: false
    end
  end
end

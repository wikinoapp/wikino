# frozen_string_literal: true

module Types
  module Object
    class NoteType < Types::Object::Base
      implements GraphQL::Types::Relay::Node

      field :number, GraphQL::Types::BigInt, null: false
      field :title, String, null: false
      field :body, String, null: false
    end
  end
end

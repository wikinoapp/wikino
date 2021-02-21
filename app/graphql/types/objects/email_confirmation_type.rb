# frozen_string_literal: true

module Types
  module Objects
    class EmailConfirmationType < Types::Objects::Base
      implements GraphQL::Types::Relay::Node

      global_id_field :id

      field :email, String, null: false
    end
  end
end

# frozen_string_literal: true

module Types
  module Object
    class UserType < Types::Object::Base
      implements GraphQL::Types::Relay::Node
    end
  end
end

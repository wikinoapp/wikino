# frozen_string_literal: true

module Types
  module InputObject
    class Base < GraphQL::Schema::InputObject
      argument_class Types::Argument::Base
    end
  end
end

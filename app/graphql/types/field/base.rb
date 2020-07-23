# frozen_string_literal: true

module Types
  module Field
    class Base < GraphQL::Schema::Field
      argument_class Types::Argument::Base
    end
  end
end

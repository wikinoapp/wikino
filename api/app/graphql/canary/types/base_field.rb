# frozen_string_literal: true

module Canary
  module Types
    class BaseField < GraphQL::Schema::Field
      argument_class Canary::Types::BaseArgument
    end
  end
end

# frozen_string_literal: true

module Canary
  module Types
    class BaseInputObject < GraphQL::Schema::InputObject
      argument_class Canary::Types::BaseArgument
    end
  end
end

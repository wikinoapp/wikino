# frozen_string_literal: true

module Canary
  module Types
    class BaseObject < GraphQL::Schema::Object
      field_class Canary::Types::BaseField
    end
  end
end

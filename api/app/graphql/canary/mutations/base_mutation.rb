# frozen_string_literal: true

module Canary
  module Mutations
    class BaseMutation < GraphQL::Schema::RelayClassicMutation
      argument_class Canary::Types::BaseArgument
      field_class Canary::Types::BaseField
      input_object_class Canary::Types::BaseInputObject
      object_class Canary::Types::BaseObject
    end
  end
end

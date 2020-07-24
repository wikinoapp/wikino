# frozen_string_literal: true

module Mutations
  class Base < GraphQL::Schema::RelayClassicMutation
    argument_class Types::Argument::Base
    field_class Types::Field::Base
    input_object_class Types::InputObject::Base
    object_class Types::Object::Base
  end
end

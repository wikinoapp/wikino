# frozen_string_literal: true

module Mutations
  class Base < GraphQL::Schema::RelayClassicMutation
    argument_class Types::Arguments::Base
    field_class Types::Fields::Base
    input_object_class Types::InputObjects::Base
    object_class Types::Objects::Base
  end
end

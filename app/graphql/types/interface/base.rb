# frozen_string_literal: true

module Types
  module Interface
    module Base
      include GraphQL::Schema::Interface

      field_class Types::Field::Base
    end
  end
end

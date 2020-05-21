# frozen_string_literal: true

module Canary
  module Types
    module BaseInterface
      include GraphQL::Schema::Interface

      field_class Canary::Types::BaseField
    end
  end
end
